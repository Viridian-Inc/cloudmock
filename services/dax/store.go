package dax

import (
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Cluster represents a DAX cluster.
type Cluster struct {
	ClusterName           string
	ClusterArn            string
	Description           string
	NodeType              string
	ReplicationFactor     int
	Status                string // creating, available, modifying, deleting
	SubnetGroupName       string
	ParameterGroupName    string
	SecurityGroupIds      []string
	AvailabilityZones     []string
	Nodes                 []*Node
	PreferredMaintenanceWindow string
	NotificationTopicArn  string
	IamRoleArn            string
	SSEDescription        *SSEDescription
	Tags                  map[string]string
	CreateTime            time.Time
	Endpoint              *Endpoint
	lifecycle             *lifecycle.Machine
}

// Node represents a single DAX node.
type Node struct {
	NodeId                    string
	Endpoint                  *Endpoint
	NodeCreateTime            time.Time
	AvailabilityZone          string
	NodeStatus                string
	ParameterGroupStatus      string
}

// Endpoint represents a DAX endpoint.
type Endpoint struct {
	Address string
	Port    int
	URL     string
}

// SSEDescription describes server-side encryption for a cluster.
type SSEDescription struct {
	Status string // ENABLED, DISABLED
}

// SubnetGroup represents a DAX subnet group.
type SubnetGroup struct {
	SubnetGroupName string
	Description     string
	VpcId           string
	Subnets         []string
}

// ParameterGroup represents a DAX parameter group.
type ParameterGroup struct {
	ParameterGroupName string
	Description        string
	Parameters         map[string]string
}

// Parameter represents a single DAX parameter.
type Parameter struct {
	ParameterName  string
	ParameterValue string
	DataType       string
	IsModifiable   string
	Description    string
}

// Store manages all DAX state in memory.
type Store struct {
	mu              sync.RWMutex
	clusters        map[string]*Cluster
	subnetGroups    map[string]*SubnetGroup
	parameterGroups map[string]*ParameterGroup
	accountID       string
	region          string
	lcConfig        *lifecycle.Config
}

// NewStore returns a new empty DAX Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*Cluster),
		subnetGroups:    make(map[string]*SubnetGroup),
		parameterGroups: make(map[string]*ParameterGroup),
		accountID:       accountID,
		region:          region,
		lcConfig:        lifecycle.DefaultConfig(),
	}
}

func (s *Store) clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:dax:%s:%s:cache/%s", s.region, s.accountID, name)
}

func buildNodes(clusterName, nodeType string, count int, azs []string, region string) []*Node {
	nodes := make([]*Node, count)
	for i := 0; i < count; i++ {
		az := region + "a"
		if i < len(azs) {
			az = azs[i]
		}
		nodeID := fmt.Sprintf("%s-%04d", clusterName, i)
		nodes[i] = &Node{
			NodeId: nodeID,
			Endpoint: &Endpoint{
				Address: fmt.Sprintf("%s.%s.nodes.dax.amazonaws.com", nodeID, region),
				Port:    8111,
			},
			NodeCreateTime:       time.Now().UTC(),
			AvailabilityZone:     az,
			NodeStatus:           "available",
			ParameterGroupStatus: "in-sync",
		}
	}
	return nodes
}

// CreateCluster creates a new DAX cluster.
func (s *Store) CreateCluster(name, description, nodeType string, replicationFactor int, subnetGroupName, parameterGroupName, iamRoleArn string, azs, securityGroupIds []string, sseEnabled bool, tags map[string]string) (*Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[name]; ok {
		return nil, fmt.Errorf("cluster already exists: %s", name)
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if nodeType == "" {
		nodeType = "dax.r4.large"
	}
	if replicationFactor < 1 {
		replicationFactor = 1
	}

	transitions := []lifecycle.Transition{
		{From: "creating", To: "available", Delay: 3 * time.Second},
	}
	lm := lifecycle.NewMachine("creating", transitions, s.lcConfig)

	clusterEndpoint := &Endpoint{
		Address: fmt.Sprintf("clusters.%s.dax.amazonaws.com", name),
		Port:    8111,
		URL:     fmt.Sprintf("daxs://clusters.%s.dax.amazonaws.com", name),
	}

	var sse *SSEDescription
	if sseEnabled {
		sse = &SSEDescription{Status: "ENABLED"}
	} else {
		sse = &SSEDescription{Status: "DISABLED"}
	}

	cluster := &Cluster{
		ClusterName:       name,
		ClusterArn:        s.clusterARN(name),
		Description:       description,
		NodeType:          nodeType,
		ReplicationFactor: replicationFactor,
		Status:            "creating",
		SubnetGroupName:   subnetGroupName,
		ParameterGroupName: parameterGroupName,
		IamRoleArn:        iamRoleArn,
		SecurityGroupIds:  securityGroupIds,
		AvailabilityZones: azs,
		Nodes:             buildNodes(name, nodeType, replicationFactor, azs, s.region),
		SSEDescription:    sse,
		Tags:              tags,
		CreateTime:        time.Now().UTC(),
		Endpoint:          clusterEndpoint,
		lifecycle:         lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		cluster.Status = string(to)
	})
	cluster.Status = string(lm.State())

	s.clusters[name] = cluster
	return cluster, nil
}

// DescribeClusters returns clusters, optionally filtered by names.
func (s *Store) DescribeClusters(names []string) []*Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*Cluster
	if len(names) == 0 {
		for _, c := range s.clusters {
			c.Status = string(c.lifecycle.State())
			result = append(result, c)
		}
		return result
	}
	for _, name := range names {
		if c, ok := s.clusters[name]; ok {
			c.Status = string(c.lifecycle.State())
			result = append(result, c)
		}
	}
	return result
}

// UpdateCluster updates a cluster's description and settings.
func (s *Store) UpdateCluster(name, description, preferredMaintenanceWindow, notificationTopicArn string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok {
		return nil, false
	}
	if description != "" {
		c.Description = description
	}
	if preferredMaintenanceWindow != "" {
		c.PreferredMaintenanceWindow = preferredMaintenanceWindow
	}
	if notificationTopicArn != "" {
		c.NotificationTopicArn = notificationTopicArn
	}
	c.Status = "modifying"
	return c, true
}

// DeleteCluster removes a cluster.
func (s *Store) DeleteCluster(name string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok {
		return nil, false
	}
	c.Status = "deleting"
	delete(s.clusters, name)
	return c, true
}

// IncreaseReplicationFactor adds nodes to a cluster.
func (s *Store) IncreaseReplicationFactor(name string, newCount int, azs []string) (*Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", name)
	}
	if newCount <= c.ReplicationFactor {
		return nil, fmt.Errorf("new replication factor %d must be greater than current %d", newCount, c.ReplicationFactor)
	}
	addCount := newCount - c.ReplicationFactor
	c.Nodes = append(c.Nodes, buildNodes(name+"-new", c.NodeType, addCount, azs, s.region)...)
	c.ReplicationFactor = newCount
	return c, nil
}

// DecreaseReplicationFactor removes nodes from a cluster.
func (s *Store) DecreaseReplicationFactor(name string, newCount int, nodeIDsToRemove []string) (*Cluster, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.clusters[name]
	if !ok {
		return nil, fmt.Errorf("cluster not found: %s", name)
	}
	if newCount >= c.ReplicationFactor {
		return nil, fmt.Errorf("new replication factor %d must be less than current %d", newCount, c.ReplicationFactor)
	}
	if newCount < 1 {
		return nil, fmt.Errorf("replication factor must be at least 1")
	}
	c.Nodes = c.Nodes[:newCount]
	c.ReplicationFactor = newCount
	return c, nil
}

// CreateSubnetGroup creates a new subnet group.
func (s *Store) CreateSubnetGroup(name, description, vpcID string, subnets []string) (*SubnetGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; ok {
		return nil, fmt.Errorf("subnet group already exists: %s", name)
	}
	sg := &SubnetGroup{
		SubnetGroupName: name,
		Description:     description,
		VpcId:           vpcID,
		Subnets:         subnets,
	}
	s.subnetGroups[name] = sg
	return sg, nil
}

// DescribeSubnetGroups returns subnet groups, optionally filtered by names.
func (s *Store) DescribeSubnetGroups(names []string) []*SubnetGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*SubnetGroup
	if len(names) == 0 {
		for _, sg := range s.subnetGroups {
			result = append(result, sg)
		}
		return result
	}
	for _, name := range names {
		if sg, ok := s.subnetGroups[name]; ok {
			result = append(result, sg)
		}
	}
	return result
}

// DeleteSubnetGroup removes a subnet group.
func (s *Store) DeleteSubnetGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.subnetGroups[name]; !ok {
		return false
	}
	delete(s.subnetGroups, name)
	return true
}

// CreateParameterGroup creates a new parameter group.
func (s *Store) CreateParameterGroup(name, description string) (*ParameterGroup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; ok {
		return nil, fmt.Errorf("parameter group already exists: %s", name)
	}
	pg := &ParameterGroup{
		ParameterGroupName: name,
		Description:        description,
		Parameters:         make(map[string]string),
	}
	s.parameterGroups[name] = pg
	return pg, nil
}

// DescribeParameterGroups returns parameter groups, optionally filtered by names.
func (s *Store) DescribeParameterGroups(names []string) []*ParameterGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []*ParameterGroup
	if len(names) == 0 {
		for _, pg := range s.parameterGroups {
			result = append(result, pg)
		}
		return result
	}
	for _, name := range names {
		if pg, ok := s.parameterGroups[name]; ok {
			result = append(result, pg)
		}
	}
	return result
}

// UpdateParameterGroup updates parameters in a parameter group.
func (s *Store) UpdateParameterGroup(name string, params map[string]string) (*ParameterGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	pg, ok := s.parameterGroups[name]
	if !ok {
		return nil, false
	}
	for k, v := range params {
		pg.Parameters[k] = v
	}
	return pg, true
}

// DeleteParameterGroup removes a parameter group.
func (s *Store) DeleteParameterGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.parameterGroups[name]; !ok {
		return false
	}
	delete(s.parameterGroups, name)
	return true
}

// DescribeParameters returns the parameters for a given parameter group.
func (s *Store) DescribeParameters(groupName string) ([]*Parameter, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	pg, ok := s.parameterGroups[groupName]
	if !ok {
		return nil, false
	}
	result := make([]*Parameter, 0, len(pg.Parameters))
	for k, v := range pg.Parameters {
		result = append(result, &Parameter{
			ParameterName:  k,
			ParameterValue: v,
			DataType:       "string",
			IsModifiable:   "TRUE",
		})
	}
	return result, true
}

// DescribeDefaultParameters returns the default DAX parameters.
func (s *Store) DescribeDefaultParameters() []*Parameter {
	return []*Parameter{
		{ParameterName: "query-ttl-millis", ParameterValue: "300000", DataType: "integer", IsModifiable: "TRUE", Description: "TTL for cached query results in milliseconds"},
		{ParameterName: "record-ttl-millis", ParameterValue: "300000", DataType: "integer", IsModifiable: "TRUE", Description: "TTL for cached item records in milliseconds"},
	}
}

// TagResource adds tags to a cluster.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clusters {
		if c.ClusterArn == arn {
			for k, v := range tags {
				c.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tags from a cluster.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, c := range s.clusters {
		if c.ClusterArn == arn {
			for _, k := range keys {
				delete(c.Tags, k)
			}
			return true
		}
	}
	return false
}

// ListTags returns tags for a cluster by ARN.
func (s *Store) ListTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clusters {
		if c.ClusterArn == arn {
			cp := make(map[string]string, len(c.Tags))
			for k, v := range c.Tags {
				cp[k] = v
			}
			return cp, true
		}
	}
	return nil, false
}
