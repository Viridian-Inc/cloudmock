package elasticache

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// CacheNode represents a single cache node within a cluster.
type CacheNode struct {
	CacheNodeID    string
	CacheNodeStatus string
	Endpoint       *Endpoint
	CreateTime     time.Time
}

// CacheCluster represents an ElastiCache cache cluster.
type CacheCluster struct {
	ID                  string
	ARN                 string
	Engine              string
	EngineVersion       string
	CacheNodeType       string
	NumCacheNodes       int
	Status              string
	PreferredAZ         string
	CacheSubnetGroupName string
	CacheParameterGroupName string
	SecurityGroupIDs    []string
	Port                int
	Endpoint            *Endpoint
	ReplicationGroupID  string
	CreatedTime         time.Time
	Tags                map[string]string
	Lifecycle           *lifecycle.Machine
	CacheNodes          []*CacheNode
}

// Endpoint holds address and port for an ElastiCache resource.
type Endpoint struct {
	Address string
	Port    int
}

// ReplicationGroup represents an ElastiCache replication group.
type ReplicationGroup struct {
	ID                  string
	ARN                 string
	Description         string
	Status              string
	Engine              string
	EngineVersion       string
	CacheNodeType       string
	NumCacheClusters    int
	AutomaticFailover   string // enabled, disabled
	MultiAZEnabled      bool
	MemberClusters      []string
	PrimaryEndpoint     *Endpoint
	ReaderEndpoint      *Endpoint
	Port                int
	CacheSubnetGroupName string
	Tags                map[string]string
	Lifecycle           *lifecycle.Machine
}

// CacheSubnetGroup represents an ElastiCache subnet group.
type CacheSubnetGroup struct {
	Name        string
	ARN         string
	Description string
	VpcID       string
	SubnetIDs   []string
}

// CacheParameterGroup represents an ElastiCache parameter group.
type CacheParameterGroup struct {
	Name        string
	ARN         string
	Family      string
	Description string
}

// Store manages all ElastiCache resources.
type Store struct {
	mu               sync.RWMutex
	clusters         map[string]*CacheCluster       // keyed by ID
	replicationGroups map[string]*ReplicationGroup   // keyed by ID
	subnetGroups     map[string]*CacheSubnetGroup    // keyed by name
	parameterGroups  map[string]*CacheParameterGroup // keyed by name
	snapshots        map[string]*Snapshot            // keyed by name
	accountID        string
	region           string
	lifecycleCfg     *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:          make(map[string]*CacheCluster),
		replicationGroups: make(map[string]*ReplicationGroup),
		subnetGroups:      make(map[string]*CacheSubnetGroup),
		parameterGroups:   make(map[string]*CacheParameterGroup),
		snapshots:         make(map[string]*Snapshot),
		accountID:         accountID,
		region:            region,
		lifecycleCfg:      lifecycle.DefaultConfig(),
	}
}

// ---- ARN helpers ----

func (s *Store) clusterARN(id string) string {
	return fmt.Sprintf("arn:aws:elasticache:%s:%s:cluster:%s", s.region, s.accountID, id)
}

func (s *Store) replicationGroupARN(id string) string {
	return fmt.Sprintf("arn:aws:elasticache:%s:%s:replicationgroup:%s", s.region, s.accountID, id)
}

func (s *Store) subnetGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:elasticache:%s:%s:subnetgroup:%s", s.region, s.accountID, name)
}

func (s *Store) parameterGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:elasticache:%s:%s:parametergroup:%s", s.region, s.accountID, name)
}

func (s *Store) clusterEndpoint(id string, port int) *Endpoint {
	return &Endpoint{
		Address: fmt.Sprintf("%s.%s.%s.cache.amazonaws.com", id, randomHex(6), s.region),
		Port:    port,
	}
}

func defaultPort(engine string) int {
	if engine == "memcached" {
		return 11211
	}
	return 6379
}

// ---- CacheCluster operations ----

func (s *Store) CreateCacheCluster(id, engine, engineVersion, nodeType, az, subnetGroup, paramGroup string, numNodes int, sgIDs []string) (*CacheCluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clusters[id]; exists {
		return nil, false
	}

	if engine == "" {
		engine = "redis"
	}
	if nodeType == "" {
		nodeType = "cache.t3.micro"
	}
	if numNodes == 0 {
		numNodes = 1
	}

	port := defaultPort(engine)

	transitions := []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("creating", transitions, s.lifecycleCfg)

	now := time.Now().UTC()
	nodes := make([]*CacheNode, numNodes)
	for i := 0; i < numNodes; i++ {
		nodeID := fmt.Sprintf("%04d", i+1)
		nodes[i] = &CacheNode{
			CacheNodeID:     nodeID,
			CacheNodeStatus: "creating",
			Endpoint:        s.clusterEndpoint(fmt.Sprintf("%s-%s", id, nodeID), port),
			CreateTime:      now,
		}
	}

	cc := &CacheCluster{
		ID:                      id,
		ARN:                     s.clusterARN(id),
		Engine:                  engine,
		EngineVersion:           engineVersion,
		CacheNodeType:           nodeType,
		NumCacheNodes:           numNodes,
		Status:                  "creating",
		PreferredAZ:             az,
		CacheSubnetGroupName:    subnetGroup,
		CacheParameterGroupName: paramGroup,
		SecurityGroupIDs:        sgIDs,
		Port:                    port,
		Endpoint:                s.clusterEndpoint(id, port),
		CreatedTime:             now,
		Tags:                    make(map[string]string),
		Lifecycle:               lm,
		CacheNodes:              nodes,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if c, ok := s.clusters[id]; ok {
			c.Status = string(to)
			for _, node := range c.CacheNodes {
				node.CacheNodeStatus = string(to)
			}
		}
	})

	s.clusters[id] = cc
	return cc, true
}

func (s *Store) GetCacheCluster(id string) (*CacheCluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cc, ok := s.clusters[id]
	return cc, ok
}

func (s *Store) ListCacheClusters(filterID string) []*CacheCluster {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CacheCluster, 0, len(s.clusters))
	for _, cc := range s.clusters {
		if filterID == "" || cc.ID == filterID {
			result = append(result, cc)
		}
	}
	return result
}

func (s *Store) ModifyCacheCluster(id, nodeType, engineVersion string, numNodes int) (*CacheCluster, bool) {
	s.mu.Lock()

	cc, ok := s.clusters[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	if nodeType != "" {
		cc.CacheNodeType = nodeType
	}
	if engineVersion != "" {
		cc.EngineVersion = engineVersion
	}
	if numNodes > 0 {
		cc.NumCacheNodes = numNodes
	}
	cc.Status = "modifying"
	lc := cc.Lifecycle
	s.mu.Unlock()

	if lc != nil {
		lc.ForceState("modifying")
	}
	return cc, true
}

func (s *Store) DeleteCacheCluster(id string) (*CacheCluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cc, ok := s.clusters[id]
	if !ok {
		return nil, false
	}
	cc.Status = "deleting"
	if cc.Lifecycle != nil {
		cc.Lifecycle.Stop()
	}
	delete(s.clusters, id)
	return cc, true
}

// ---- ReplicationGroup operations ----

func (s *Store) CreateReplicationGroup(id, description, engine, engineVersion, nodeType, subnetGroup, failover string, numClusters int, multiAZ bool, port int) (*ReplicationGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.replicationGroups[id]; exists {
		return nil, false
	}

	if engine == "" {
		engine = "redis"
	}
	if nodeType == "" {
		nodeType = "cache.t3.micro"
	}
	if numClusters == 0 {
		numClusters = 1
	}
	if failover == "" {
		failover = "disabled"
	}
	if port == 0 {
		port = defaultPort(engine)
	}

	transitions := []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("creating", transitions, s.lifecycleCfg)

	members := make([]string, 0, numClusters)
	for i := 1; i <= numClusters; i++ {
		members = append(members, fmt.Sprintf("%s-%03d", id, i))
	}

	rg := &ReplicationGroup{
		ID:                   id,
		ARN:                  s.replicationGroupARN(id),
		Description:          description,
		Status:               "creating",
		Engine:               engine,
		EngineVersion:        engineVersion,
		CacheNodeType:        nodeType,
		NumCacheClusters:     numClusters,
		AutomaticFailover:    failover,
		MultiAZEnabled:       multiAZ,
		MemberClusters:       members,
		PrimaryEndpoint:      s.clusterEndpoint(id, port),
		ReaderEndpoint:       s.clusterEndpoint(id+"-ro", port),
		Port:                 port,
		CacheSubnetGroupName: subnetGroup,
		Tags:                 make(map[string]string),
		Lifecycle:            lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if r, ok := s.replicationGroups[id]; ok {
			r.Status = string(to)
		}
	})

	s.replicationGroups[id] = rg
	return rg, true
}

func (s *Store) GetReplicationGroup(id string) (*ReplicationGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rg, ok := s.replicationGroups[id]
	return rg, ok
}

func (s *Store) ListReplicationGroups(filterID string) []*ReplicationGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*ReplicationGroup, 0, len(s.replicationGroups))
	for _, rg := range s.replicationGroups {
		if filterID == "" || rg.ID == filterID {
			result = append(result, rg)
		}
	}
	return result
}

func (s *Store) ModifyReplicationGroup(id, description, nodeType, engineVersion, failover string) (*ReplicationGroup, bool) {
	s.mu.Lock()

	rg, ok := s.replicationGroups[id]
	if !ok {
		s.mu.Unlock()
		return nil, false
	}
	if description != "" {
		rg.Description = description
	}
	if nodeType != "" {
		rg.CacheNodeType = nodeType
	}
	if engineVersion != "" {
		rg.EngineVersion = engineVersion
	}
	if failover != "" {
		rg.AutomaticFailover = failover
	}
	rg.Status = "modifying"
	lc := rg.Lifecycle
	s.mu.Unlock()

	if lc != nil {
		lc.ForceState("modifying")
	}
	return rg, true
}

func (s *Store) DeleteReplicationGroup(id string) (*ReplicationGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rg, ok := s.replicationGroups[id]
	if !ok {
		return nil, false
	}
	rg.Status = "deleting"
	if rg.Lifecycle != nil {
		rg.Lifecycle.Stop()
	}
	delete(s.replicationGroups, id)
	return rg, true
}

// ---- CacheSubnetGroup operations ----

func (s *Store) CreateCacheSubnetGroup(name, description, vpcID string, subnetIDs []string) (*CacheSubnetGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.subnetGroups[name]; exists {
		return nil, false
	}

	sg := &CacheSubnetGroup{
		Name:        name,
		ARN:         s.subnetGroupARN(name),
		Description: description,
		VpcID:       vpcID,
		SubnetIDs:   subnetIDs,
	}
	s.subnetGroups[name] = sg
	return sg, true
}

func (s *Store) GetCacheSubnetGroup(name string) (*CacheSubnetGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sg, ok := s.subnetGroups[name]
	return sg, ok
}

func (s *Store) ListCacheSubnetGroups(filterName string) []*CacheSubnetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CacheSubnetGroup, 0, len(s.subnetGroups))
	for _, sg := range s.subnetGroups {
		if filterName == "" || sg.Name == filterName {
			result = append(result, sg)
		}
	}
	return result
}

func (s *Store) DeleteCacheSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.subnetGroups[name]; !ok {
		return false
	}
	delete(s.subnetGroups, name)
	return true
}

// ---- CacheParameterGroup operations ----

func (s *Store) CreateCacheParameterGroup(name, family, description string) (*CacheParameterGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.parameterGroups[name]; exists {
		return nil, false
	}

	pg := &CacheParameterGroup{
		Name:        name,
		ARN:         s.parameterGroupARN(name),
		Family:      family,
		Description: description,
	}
	s.parameterGroups[name] = pg
	return pg, true
}

func (s *Store) GetCacheParameterGroup(name string) (*CacheParameterGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pg, ok := s.parameterGroups[name]
	return pg, ok
}

func (s *Store) ListCacheParameterGroups(filterName string) []*CacheParameterGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*CacheParameterGroup, 0, len(s.parameterGroups))
	for _, pg := range s.parameterGroups {
		if filterName == "" || pg.Name == filterName {
			result = append(result, pg)
		}
	}
	return result
}

func (s *Store) DeleteCacheParameterGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.parameterGroups[name]; !ok {
		return false
	}
	delete(s.parameterGroups, name)
	return true
}

// ---- Snapshot operations ----

// Snapshot represents an ElastiCache backup snapshot.
type Snapshot struct {
	SnapshotName        string
	ARN                 string
	CacheClusterID      string
	ReplicationGroupID  string
	Status              string
	Engine              string
	EngineVersion       string
	CacheNodeType       string
	NumCacheNodes       int
	Port                int
	Tags                map[string]string
	CreatedTime         time.Time
}

func (s *Store) snapshotARN(name string) string {
	return fmt.Sprintf("arn:aws:elasticache:%s:%s:snapshot:%s", s.region, s.accountID, name)
}

func (s *Store) CreateSnapshot(name, clusterID, replicationGroupID string, tags map[string]string) (*Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.snapshots[name]; exists {
		return nil, fmt.Errorf("already_exists")
	}

	snap := &Snapshot{
		SnapshotName:       name,
		ARN:                s.snapshotARN(name),
		Status:             "available",
		Tags:               tags,
		CreatedTime:        time.Now().UTC(),
	}

	if clusterID != "" {
		cc, ok := s.clusters[clusterID]
		if !ok {
			return nil, fmt.Errorf("not_found")
		}
		snap.CacheClusterID = clusterID
		snap.Engine = cc.Engine
		snap.EngineVersion = cc.EngineVersion
		snap.CacheNodeType = cc.CacheNodeType
		snap.NumCacheNodes = cc.NumCacheNodes
		snap.Port = cc.Port
	} else if replicationGroupID != "" {
		rg, ok := s.replicationGroups[replicationGroupID]
		if !ok {
			return nil, fmt.Errorf("not_found")
		}
		snap.ReplicationGroupID = replicationGroupID
		snap.Engine = rg.Engine
		snap.EngineVersion = rg.EngineVersion
		snap.CacheNodeType = rg.CacheNodeType
		snap.Port = rg.Port
	}

	s.snapshots[name] = snap
	return snap, nil
}

func (s *Store) ListSnapshots(filterName, filterCluster, filterRG string) []*Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]*Snapshot, 0, len(s.snapshots))
	for _, snap := range s.snapshots {
		if filterName != "" && snap.SnapshotName != filterName {
			continue
		}
		if filterCluster != "" && snap.CacheClusterID != filterCluster {
			continue
		}
		if filterRG != "" && snap.ReplicationGroupID != filterRG {
			continue
		}
		result = append(result, snap)
	}
	return result
}

func (s *Store) DeleteSnapshot(name string) (*Snapshot, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	snap, ok := s.snapshots[name]
	if !ok {
		return nil, false
	}
	delete(s.snapshots, name)
	return snap, true
}

// TestFailover simulates a failover for a replication group node group.
func (s *Store) TestFailover(id, nodeGroupID string) (*ReplicationGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rg, ok := s.replicationGroups[id]
	if !ok {
		return nil, false
	}

	// Simulate failover: swap primary and first replica in member clusters.
	if len(rg.MemberClusters) >= 2 {
		rg.MemberClusters[0], rg.MemberClusters[1] = rg.MemberClusters[1], rg.MemberClusters[0]
	}

	return rg, true
}

// ---- Tag operations ----

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

func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
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

func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, cc := range s.clusters {
		if cc.ARN == arn {
			return cc.Tags
		}
	}
	for _, rg := range s.replicationGroups {
		if rg.ARN == arn {
			return rg.Tags
		}
	}
	for _, snap := range s.snapshots {
		if snap.ARN == arn {
			return snap.Tags
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
