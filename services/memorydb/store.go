package memorydb

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Cluster represents a MemoryDB cluster.
type Cluster struct {
	Name            string
	ARN             string
	Status          string
	NodeType        string
	NumShards       int
	NumReplicasPerShard int
	EngineVersion   string
	Port            int
	ClusterEndpoint ClusterEndpoint
	SubnetGroupName string
	ACLName         string
	ParameterGroupName string
	Tags            map[string]string
	CreatedTime     time.Time
	Lifecycle       *lifecycle.Machine
}

// ClusterEndpoint holds cluster connection info.
type ClusterEndpoint struct {
	Address string
	Port    int
}

// ACL represents a MemoryDB access control list.
type ACL struct {
	Name      string
	ARN       string
	Status    string
	UserNames []string
	Tags      map[string]string
}

// User represents a MemoryDB user.
type User struct {
	Name            string
	ARN             string
	Status          string
	AccessString    string
	Authentication  AuthenticationType
	Tags            map[string]string
}

// AuthenticationType holds user auth config.
type AuthenticationType struct {
	Type      string
	Passwords []string
}

// ParameterGroup represents a MemoryDB parameter group.
type ParameterGroup struct {
	Name        string
	ARN         string
	Family      string
	Description string
	Tags        map[string]string
}

// SubnetGroup represents a MemoryDB subnet group.
type SubnetGroup struct {
	Name        string
	ARN         string
	Description string
	SubnetIds   []string
	Tags        map[string]string
}

// Snapshot represents a MemoryDB snapshot.
type Snapshot struct {
	Name         string
	ARN          string
	ClusterName  string
	Status       string
	Source       string
	Tags         map[string]string
	CreatedTime  time.Time
}

// Store manages all MemoryDB resources.
type Store struct {
	mu              sync.RWMutex
	clusters        map[string]*Cluster
	acls            map[string]*ACL
	users           map[string]*User
	parameterGroups map[string]*ParameterGroup
	subnetGroups    map[string]*SubnetGroup
	snapshots       map[string]*Snapshot
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore creates a new MemoryDB store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*Cluster),
		acls:            make(map[string]*ACL),
		users:           make(map[string]*User),
		parameterGroups: make(map[string]*ParameterGroup),
		subnetGroups:    make(map[string]*SubnetGroup),
		snapshots:       make(map[string]*Snapshot),
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

func (s *Store) clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:cluster/%s", s.region, s.accountID, name)
}
func (s *Store) aclARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:acl/%s", s.region, s.accountID, name)
}
func (s *Store) userARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:user/%s", s.region, s.accountID, name)
}
func (s *Store) parameterGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:parametergroup/%s", s.region, s.accountID, name)
}
func (s *Store) subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:subnetgroup/%s", s.region, s.accountID, name)
}
func (s *Store) snapshotARN(name string) string {
	return fmt.Sprintf("arn:aws:memorydb:%s:%s:snapshot/%s", s.region, s.accountID, name)
}

func clusterTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 5 * time.Second},
		{From: "deleting", To: "deleted", Delay: 3 * time.Second},
	}
}

// ---- Cluster operations ----

func (s *Store) CreateCluster(name, nodeType, engineVersion, subnetGroup, aclName, paramGroup string, numShards, numReplicas int, tags map[string]string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.clusters[name]; ok {
		return nil, false
	}
	if nodeType == "" { nodeType = "db.r6g.large" }
	if numShards == 0 { numShards = 1 }
	c := &Cluster{
		Name: name, ARN: s.clusterARN(name), Status: "creating",
		NodeType: nodeType, NumShards: numShards, NumReplicasPerShard: numReplicas,
		EngineVersion: engineVersion, Port: 6379,
		ClusterEndpoint: ClusterEndpoint{
			Address: fmt.Sprintf("%s.%s.%s.memorydb.amazonaws.com", name, randomHex(6), s.region),
			Port:    6379,
		},
		SubnetGroupName: subnetGroup, ACLName: aclName, ParameterGroupName: paramGroup,
		Tags: tags, CreatedTime: time.Now().UTC(),
		Lifecycle: lifecycle.NewMachine("creating", clusterTransitions(), s.lcConfig),
	}
	s.clusters[name] = c
	return c, true
}

func (s *Store) GetCluster(name string) (*Cluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[name]
	if ok { c.Status = string(c.Lifecycle.State()) }
	return c, ok
}

func (s *Store) ListClusters() []*Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Cluster, 0, len(s.clusters))
	for _, c := range s.clusters {
		c.Status = string(c.Lifecycle.State())
		result = append(result, c)
	}
	return result
}

func (s *Store) UpdateCluster(name, nodeType, engineVersion string, numShards, numReplicas int) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok { return nil, false }
	if nodeType != "" { c.NodeType = nodeType }
	if engineVersion != "" { c.EngineVersion = engineVersion }
	if numShards > 0 { c.NumShards = numShards }
	if numReplicas >= 0 { c.NumReplicasPerShard = numReplicas }
	return c, true
}

func (s *Store) DeleteCluster(name string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok { return nil, false }
	c.Lifecycle.ForceState("deleting")
	c.Status = "deleting"
	return c, true
}

// ---- ACL operations ----

func (s *Store) CreateACL(name string, userNames []string, tags map[string]string) (*ACL, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.acls[name]; ok { return nil, false }
	acl := &ACL{Name: name, ARN: s.aclARN(name), Status: "active", UserNames: userNames, Tags: tags}
	s.acls[name] = acl
	return acl, true
}

func (s *Store) GetACL(name string) (*ACL, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	acl, ok := s.acls[name]
	return acl, ok
}

func (s *Store) ListACLs() []*ACL {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ACL, 0, len(s.acls))
	for _, acl := range s.acls { result = append(result, acl) }
	return result
}

func (s *Store) UpdateACL(name string, userNamesToAdd, userNamesToRemove []string) (*ACL, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	acl, ok := s.acls[name]
	if !ok { return nil, false }
	removeSet := make(map[string]bool)
	for _, u := range userNamesToRemove { removeSet[u] = true }
	filtered := make([]string, 0)
	for _, u := range acl.UserNames {
		if !removeSet[u] { filtered = append(filtered, u) }
	}
	acl.UserNames = append(filtered, userNamesToAdd...)
	return acl, true
}

func (s *Store) DeleteACL(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.acls[name]; !ok { return false }
	delete(s.acls, name)
	return true
}

// ---- User operations ----

func (s *Store) CreateUser(name, accessString, authType string, passwords []string, tags map[string]string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[name]; ok { return nil, false }
	if authType == "" { authType = "password" }
	u := &User{
		Name: name, ARN: s.userARN(name), Status: "active",
		AccessString: accessString, Authentication: AuthenticationType{Type: authType, Passwords: passwords},
		Tags: tags,
	}
	s.users[name] = u
	return u, true
}

func (s *Store) GetUser(name string) (*User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	u, ok := s.users[name]
	return u, ok
}

func (s *Store) ListUsers() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*User, 0, len(s.users))
	for _, u := range s.users { result = append(result, u) }
	return result
}

func (s *Store) UpdateUser(name, accessString string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[name]
	if !ok { return nil, false }
	if accessString != "" { u.AccessString = accessString }
	return u, true
}

func (s *Store) DeleteUser(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[name]; !ok { return false }
	delete(s.users, name)
	return true
}

// ---- ParameterGroup operations ----

func (s *Store) CreateParameterGroup(name, family, description string, tags map[string]string) (*ParameterGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; ok { return nil, false }
	pg := &ParameterGroup{Name: name, ARN: s.parameterGroupARN(name), Family: family, Description: description, Tags: tags}
	s.parameterGroups[name] = pg
	return pg, true
}

func (s *Store) ListParameterGroups() []*ParameterGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ParameterGroup, 0, len(s.parameterGroups))
	for _, pg := range s.parameterGroups { result = append(result, pg) }
	return result
}

func (s *Store) DeleteParameterGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; !ok { return false }
	delete(s.parameterGroups, name)
	return true
}

// ---- SubnetGroup operations ----

func (s *Store) CreateSubnetGroup(name, description string, subnetIds []string, tags map[string]string) (*SubnetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; ok { return nil, false }
	sg := &SubnetGroup{Name: name, ARN: s.subnetGroupARN(name), Description: description, SubnetIds: subnetIds, Tags: tags}
	s.subnetGroups[name] = sg
	return sg, true
}

func (s *Store) ListSubnetGroups() []*SubnetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*SubnetGroup, 0, len(s.subnetGroups))
	for _, sg := range s.subnetGroups { result = append(result, sg) }
	return result
}

func (s *Store) DeleteSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; !ok { return false }
	delete(s.subnetGroups, name)
	return true
}

// ---- Snapshot operations ----

func (s *Store) CreateSnapshot(name, clusterName string, tags map[string]string) (*Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[name]; ok { return nil, false }
	if _, ok := s.clusters[clusterName]; !ok { return nil, false }
	snap := &Snapshot{
		Name: name, ARN: s.snapshotARN(name), ClusterName: clusterName,
		Status: "available", Source: "manual", Tags: tags, CreatedTime: time.Now().UTC(),
	}
	s.snapshots[name] = snap
	return snap, true
}

func (s *Store) ListSnapshots(clusterName string) []*Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Snapshot, 0)
	for _, snap := range s.snapshots {
		if clusterName == "" || snap.ClusterName == clusterName {
			result = append(result, snap)
		}
	}
	return result
}

func (s *Store) DeleteSnapshot(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.snapshots[name]; !ok { return false }
	delete(s.snapshots, name)
	return true
}

// ---- Tag operations ----

func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, c := range s.clusters { if c.ARN == arn { return c.Tags } }
	for _, a := range s.acls { if a.ARN == arn { return a.Tags } }
	for _, u := range s.users { if u.ARN == arn { return u.Tags } }
	for _, pg := range s.parameterGroups { if pg.ARN == arn { return pg.Tags } }
	for _, sg := range s.subnetGroups { if sg.ARN == arn { return sg.Tags } }
	for _, snap := range s.snapshots { if snap.ARN == arn { return snap.Tags } }
	return nil
}

func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	target := s.tagMapByARN(arn)
	if target == nil { return false }
	for k, v := range tags { target[k] = v }
	return true
}

func (s *Store) UntagResource(arn string, keys []string) bool {
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
