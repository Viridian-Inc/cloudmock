package redshift

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Cluster represents a Redshift cluster.
type Cluster struct {
	Identifier        string
	ARN               string
	NodeType          string
	NumberOfNodes     int
	MasterUsername    string
	DBName            string
	Status            string
	Endpoint          ClusterEndpoint
	VpcId             string
	ClusterSubnetGroup string
	ParameterGroupName string
	CreatedTime       time.Time
	Tags              map[string]string
	Lifecycle         *lifecycle.Machine
}

// ClusterEndpoint holds address and port.
type ClusterEndpoint struct {
	Address string
	Port    int
}

// ClusterSnapshot represents a manual or automated snapshot.
type ClusterSnapshot struct {
	Identifier         string
	ARN                string
	ClusterIdentifier  string
	Status             string
	NodeType           string
	NumberOfNodes      int
	DBName             string
	MasterUsername     string
	SnapshotCreateTime time.Time
	Tags               map[string]string
}

// ClusterSubnetGroup represents a Redshift subnet group.
type ClusterSubnetGroup struct {
	Name        string
	ARN         string
	Description string
	SubnetIds   []string
	VpcId       string
	Status      string
	Tags        map[string]string
}

// ClusterParameterGroup represents a Redshift parameter group.
type ClusterParameterGroup struct {
	Name        string
	ARN         string
	Family      string
	Description string
	Tags        map[string]string
}

// Store manages all Redshift resources in memory.
type Store struct {
	mu              sync.RWMutex
	clusters        map[string]*Cluster
	snapshots       map[string]*ClusterSnapshot
	subnetGroups    map[string]*ClusterSubnetGroup
	parameterGroups map[string]*ClusterParameterGroup
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore creates a new Redshift Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*Cluster),
		snapshots:       make(map[string]*ClusterSnapshot),
		subnetGroups:    make(map[string]*ClusterSubnetGroup),
		parameterGroups: make(map[string]*ClusterParameterGroup),
		accountID:       accountID,
		region:          region,
		lcConfig:        lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) clusterARN(id string) string {
	return fmt.Sprintf("arn:aws:redshift:%s:%s:cluster:%s", s.region, s.accountID, id)
}

func (s *Store) snapshotARN(id string) string {
	return fmt.Sprintf("arn:aws:redshift:%s:%s:snapshot:%s", s.region, s.accountID, id)
}

func (s *Store) subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:redshift:%s:%s:subnetgroup:%s", s.region, s.accountID, name)
}

func (s *Store) parameterGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:redshift:%s:%s:parametergroup:%s", s.region, s.accountID, name)
}

func clusterTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 5 * time.Second},
		{From: "deleting", To: "deleted", Delay: 3 * time.Second},
	}
}

func (s *Store) CreateCluster(id, nodeType string, numNodes int, masterUser, dbName, subnetGroup, paramGroup string, tags map[string]string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[id]; ok {
		return nil, false
	}
	if numNodes == 0 {
		numNodes = 1
	}
	if dbName == "" {
		dbName = "dev"
	}
	endpoint := ClusterEndpoint{
		Address: fmt.Sprintf("%s.%s.%s.redshift.amazonaws.com", id, randomHex(8), s.region),
		Port:    5439,
	}
	c := &Cluster{
		Identifier:         id,
		ARN:                s.clusterARN(id),
		NodeType:           nodeType,
		NumberOfNodes:      numNodes,
		MasterUsername:     masterUser,
		DBName:             dbName,
		Status:             "creating",
		Endpoint:           endpoint,
		ClusterSubnetGroup: subnetGroup,
		ParameterGroupName: paramGroup,
		CreatedTime:        time.Now().UTC(),
		Tags:               tags,
		Lifecycle:          lifecycle.NewMachine("creating", clusterTransitions(), s.lcConfig),
	}
	s.clusters[id] = c
	return c, true
}

func (s *Store) GetCluster(id string) (*Cluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[id]
	if ok {
		c.Status = string(c.Lifecycle.State())
	}
	return c, ok
}

func (s *Store) ListClusters(filterID string) []*Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Cluster, 0)
	for _, c := range s.clusters {
		c.Status = string(c.Lifecycle.State())
		if filterID == "" || c.Identifier == filterID {
			result = append(result, c)
		}
	}
	return result
}

func (s *Store) ModifyCluster(id, nodeType string, numNodes int) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[id]
	if !ok {
		return nil, false
	}
	if nodeType != "" {
		c.NodeType = nodeType
	}
	if numNodes > 0 {
		c.NumberOfNodes = numNodes
	}
	return c, true
}

func (s *Store) DeleteCluster(id string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[id]
	if !ok {
		return nil, false
	}
	c.Lifecycle.ForceState("deleting")
	c.Status = "deleting"
	return c, true
}

func (s *Store) RebootCluster(id string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[id]
	if !ok {
		return nil, false
	}
	// Reboot is a no-op for mock; cluster stays available.
	return c, true
}

// ---- Snapshot operations ----

func (s *Store) CreateClusterSnapshot(snapshotID, clusterID string, tags map[string]string) (*ClusterSnapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[snapshotID]; ok {
		return nil, false
	}
	c, ok := s.clusters[clusterID]
	if !ok {
		return nil, false
	}
	snap := &ClusterSnapshot{
		Identifier:         snapshotID,
		ARN:                s.snapshotARN(snapshotID),
		ClusterIdentifier:  clusterID,
		Status:             "available",
		NodeType:           c.NodeType,
		NumberOfNodes:      c.NumberOfNodes,
		DBName:             c.DBName,
		MasterUsername:     c.MasterUsername,
		SnapshotCreateTime: time.Now().UTC(),
		Tags:               tags,
	}
	s.snapshots[snapshotID] = snap
	return snap, true
}

func (s *Store) ListClusterSnapshots(clusterID, snapshotID string) []*ClusterSnapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ClusterSnapshot, 0)
	for _, snap := range s.snapshots {
		if snapshotID != "" && snap.Identifier != snapshotID {
			continue
		}
		if clusterID != "" && snap.ClusterIdentifier != clusterID {
			continue
		}
		result = append(result, snap)
	}
	return result
}

func (s *Store) DeleteClusterSnapshot(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[id]; !ok {
		return false
	}
	delete(s.snapshots, id)
	return true
}

func (s *Store) RestoreFromClusterSnapshot(newClusterID, snapshotID string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	snap, ok := s.snapshots[snapshotID]
	if !ok {
		return nil, false
	}
	if _, exists := s.clusters[newClusterID]; exists {
		return nil, false
	}
	endpoint := ClusterEndpoint{
		Address: fmt.Sprintf("%s.%s.%s.redshift.amazonaws.com", newClusterID, randomHex(8), s.region),
		Port:    5439,
	}
	c := &Cluster{
		Identifier:     newClusterID,
		ARN:            s.clusterARN(newClusterID),
		NodeType:       snap.NodeType,
		NumberOfNodes:  snap.NumberOfNodes,
		MasterUsername: snap.MasterUsername,
		DBName:         snap.DBName,
		Status:         "creating",
		Endpoint:       endpoint,
		CreatedTime:    time.Now().UTC(),
		Tags:           make(map[string]string),
		Lifecycle:      lifecycle.NewMachine("creating", clusterTransitions(), s.lcConfig),
	}
	s.clusters[newClusterID] = c
	return c, true
}

// ---- SubnetGroup operations ----

func (s *Store) CreateClusterSubnetGroup(name, description string, subnetIDs []string, tags map[string]string) (*ClusterSubnetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; ok {
		return nil, false
	}
	sg := &ClusterSubnetGroup{
		Name:        name,
		ARN:         s.subnetGroupARN(name),
		Description: description,
		SubnetIds:   subnetIDs,
		Status:      "Complete",
		Tags:        tags,
	}
	s.subnetGroups[name] = sg
	return sg, true
}

func (s *Store) ListClusterSubnetGroups(filterName string) []*ClusterSubnetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ClusterSubnetGroup, 0)
	for _, sg := range s.subnetGroups {
		if filterName == "" || sg.Name == filterName {
			result = append(result, sg)
		}
	}
	return result
}

func (s *Store) DeleteClusterSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; !ok {
		return false
	}
	delete(s.subnetGroups, name)
	return true
}

// ---- ParameterGroup operations ----

func (s *Store) CreateClusterParameterGroup(name, family, description string, tags map[string]string) (*ClusterParameterGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; ok {
		return nil, false
	}
	pg := &ClusterParameterGroup{
		Name:        name,
		ARN:         s.parameterGroupARN(name),
		Family:      family,
		Description: description,
		Tags:        tags,
	}
	s.parameterGroups[name] = pg
	return pg, true
}

func (s *Store) ListClusterParameterGroups(filterName string) []*ClusterParameterGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ClusterParameterGroup, 0)
	for _, pg := range s.parameterGroups {
		if filterName == "" || pg.Name == filterName {
			result = append(result, pg)
		}
	}
	return result
}

func (s *Store) DeleteClusterParameterGroup(name string) bool {
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
	for _, pg := range s.parameterGroups {
		if pg.ARN == arn {
			return pg.Tags
		}
	}
	return nil
}

func (s *Store) AddTags(arn string, tags map[string]string) bool {
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

func (s *Store) RemoveTags(arn string, keys []string) bool {
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

func (s *Store) ListTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	target := s.tagMapByARN(arn)
	if target == nil {
		return nil, false
	}
	result := make(map[string]string, len(target))
	for k, v := range target {
		result[k] = v
	}
	return result, true
}
