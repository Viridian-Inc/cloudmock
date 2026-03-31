package eks

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Cluster represents an EKS cluster.
type Cluster struct {
	Name               string
	ARN                string
	Version            string
	RoleARN            string
	Status             string
	Endpoint           string
	CertificateAuthority string
	PlatformVersion    string
	SubnetIDs          []string
	SecurityGroupIDs   []string
	ClusterSecurityGroupID string
	ServiceCIDR        string
	VPCID              string
	CreatedAt          time.Time
	Tags               map[string]string
	Lifecycle          *lifecycle.Machine
}

// Nodegroup represents an EKS managed node group.
type Nodegroup struct {
	Name              string
	ARN               string
	ClusterName       string
	NodeRole          string
	Status            string
	InstanceTypes     []string
	AmiType           string
	DiskSize          int
	ScalingConfig     *NodegroupScalingConfig
	SubnetIDs         []string
	Labels            map[string]string
	Taints            []Taint
	CapacityType      string // ON_DEMAND, SPOT
	CreatedAt         time.Time
	Tags              map[string]string
	Lifecycle         *lifecycle.Machine
}

// NodegroupScalingConfig represents scaling configuration for a node group.
type NodegroupScalingConfig struct {
	MinSize     int
	MaxSize     int
	DesiredSize int
}

// Taint represents a Kubernetes taint on a node group.
type Taint struct {
	Key    string
	Value  string
	Effect string // NO_SCHEDULE, NO_EXECUTE, PREFER_NO_SCHEDULE
}

// FargateProfile represents an EKS Fargate profile.
type FargateProfile struct {
	Name            string
	ARN             string
	ClusterName     string
	PodExecutionRoleARN string
	Status          string
	SubnetIDs       []string
	Selectors       []FargateSelector
	CreatedAt       time.Time
	Tags            map[string]string
	Lifecycle       *lifecycle.Machine
}

// FargateSelector represents a pod selector for Fargate.
type FargateSelector struct {
	Namespace string
	Labels    map[string]string
}

// Addon represents an EKS addon.
type Addon struct {
	Name           string
	ARN            string
	ClusterName    string
	AddonVersion   string
	Status         string
	ServiceAccountRoleARN string
	CreatedAt      time.Time
	ModifiedAt     time.Time
	Tags           map[string]string
	Lifecycle      *lifecycle.Machine
}

// Store manages all EKS resources.
type Store struct {
	mu              sync.RWMutex
	clusters        map[string]*Cluster           // keyed by name
	nodegroups      map[string]map[string]*Nodegroup // clusterName -> ngName -> Nodegroup
	fargateProfiles map[string]map[string]*FargateProfile // clusterName -> profileName -> Profile
	addons          map[string]map[string]*Addon  // clusterName -> addonName -> Addon
	accountID       string
	region          string
	lifecycleCfg    *lifecycle.Config
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		clusters:        make(map[string]*Cluster),
		nodegroups:      make(map[string]map[string]*Nodegroup),
		fargateProfiles: make(map[string]map[string]*FargateProfile),
		addons:          make(map[string]map[string]*Addon),
		accountID:       accountID,
		region:          region,
		lifecycleCfg:    lifecycle.DefaultConfig(),
	}
}

// ---- ARN helpers ----

func (s *Store) clusterARN(name string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:cluster/%s", s.region, s.accountID, name)
}

func (s *Store) nodegroupARN(clusterName, ngName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:nodegroup/%s/%s/%s", s.region, s.accountID, clusterName, ngName, newID())
}

func (s *Store) fargateProfileARN(clusterName, profileName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:fargateprofile/%s/%s/%s", s.region, s.accountID, clusterName, profileName, newID())
}

func (s *Store) addonARN(clusterName, addonName string) string {
	return fmt.Sprintf("arn:aws:eks:%s:%s:addon/%s/%s/%s", s.region, s.accountID, clusterName, addonName, newID())
}

func (s *Store) clusterEndpoint(name string) string {
	return fmt.Sprintf("https://%s.gr7.%s.eks.amazonaws.com", randomHex(16), s.region)
}

func (s *Store) clusterCertAuth() string {
	return fmt.Sprintf("LS0t%s==", randomHex(32))
}

// ---- Cluster operations ----

func (s *Store) CreateCluster(name, version, roleARN, vpcID, serviceCIDR string, subnetIDs, sgIDs []string, tags map[string]string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.clusters[name]; exists {
		return nil, false
	}

	if version == "" {
		version = "1.29"
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleCfg)

	c := &Cluster{
		Name:                   name,
		ARN:                    s.clusterARN(name),
		Version:                version,
		RoleARN:                roleARN,
		Status:                 "CREATING",
		Endpoint:               s.clusterEndpoint(name),
		CertificateAuthority:   s.clusterCertAuth(),
		PlatformVersion:        "eks.1",
		SubnetIDs:              subnetIDs,
		SecurityGroupIDs:       sgIDs,
		ClusterSecurityGroupID: fmt.Sprintf("sg-%s", randomHex(8)),
		ServiceCIDR:            serviceCIDR,
		VPCID:                  vpcID,
		CreatedAt:              time.Now().UTC(),
		Tags:                   tags,
		Lifecycle:              lm,
	}

	if c.Tags == nil {
		c.Tags = make(map[string]string)
	}
	if c.ServiceCIDR == "" {
		c.ServiceCIDR = "10.100.0.0/16"
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if cl, ok := s.clusters[name]; ok {
			cl.Status = string(to)
		}
	})

	s.clusters[name] = c
	s.nodegroups[name] = make(map[string]*Nodegroup)
	s.fargateProfiles[name] = make(map[string]*FargateProfile)
	s.addons[name] = make(map[string]*Addon)
	return c, true
}

func (s *Store) GetCluster(name string) (*Cluster, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.clusters[name]
	return c, ok
}

func (s *Store) ListClusters() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.clusters))
	for name := range s.clusters {
		names = append(names, name)
	}
	return names
}

func (s *Store) UpdateClusterConfig(name, version string, subnetIDs, sgIDs []string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.clusters[name]
	if !ok {
		return nil, false
	}
	if version != "" {
		c.Version = version
	}
	if len(subnetIDs) > 0 {
		c.SubnetIDs = subnetIDs
	}
	if len(sgIDs) > 0 {
		c.SecurityGroupIDs = sgIDs
	}
	c.Status = "UPDATING"
	if c.Lifecycle != nil {
		c.Lifecycle.ForceState("UPDATING")
	}
	return c, true
}

func (s *Store) DeleteCluster(name string) (*Cluster, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	c, ok := s.clusters[name]
	if !ok {
		return nil, false
	}

	// Cannot delete cluster with active nodegroups.
	if ngs, ok := s.nodegroups[name]; ok && len(ngs) > 0 {
		return nil, false
	}

	c.Status = "DELETING"
	if c.Lifecycle != nil {
		c.Lifecycle.Stop()
	}
	delete(s.clusters, name)
	delete(s.nodegroups, name)
	delete(s.fargateProfiles, name)
	delete(s.addons, name)
	return c, true
}

// ---- Nodegroup operations ----

func (s *Store) CreateNodegroup(clusterName, ngName, nodeRole, amiType, capacityType string, instanceTypes, subnetIDs []string, diskSize int, scaling *NodegroupScalingConfig, labels map[string]string, taints []Taint, tags map[string]string) (*Nodegroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, false
	}

	ngs, ok := s.nodegroups[clusterName]
	if !ok {
		ngs = make(map[string]*Nodegroup)
		s.nodegroups[clusterName] = ngs
	}
	if _, exists := ngs[ngName]; exists {
		return nil, false
	}

	if len(instanceTypes) == 0 {
		instanceTypes = []string{"t3.medium"}
	}
	if amiType == "" {
		amiType = "AL2_x86_64"
	}
	if capacityType == "" {
		capacityType = "ON_DEMAND"
	}
	if diskSize == 0 {
		diskSize = 20
	}
	if scaling == nil {
		scaling = &NodegroupScalingConfig{MinSize: 1, MaxSize: 2, DesiredSize: 1}
	}
	if tags == nil {
		tags = make(map[string]string)
	}
	if labels == nil {
		labels = make(map[string]string)
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 2 * time.Second},
	}
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleCfg)

	ng := &Nodegroup{
		Name:          ngName,
		ARN:           s.nodegroupARN(clusterName, ngName),
		ClusterName:   clusterName,
		NodeRole:      nodeRole,
		Status:        "CREATING",
		InstanceTypes: instanceTypes,
		AmiType:       amiType,
		DiskSize:      diskSize,
		ScalingConfig: scaling,
		SubnetIDs:     subnetIDs,
		Labels:        labels,
		Taints:        taints,
		CapacityType:  capacityType,
		CreatedAt:     time.Now().UTC(),
		Tags:          tags,
		Lifecycle:     lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if ngs, ok := s.nodegroups[clusterName]; ok {
			if n, ok := ngs[ngName]; ok {
				n.Status = string(to)
			}
		}
	})

	ngs[ngName] = ng
	return ng, true
}

func (s *Store) GetNodegroup(clusterName, ngName string) (*Nodegroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ngs, ok := s.nodegroups[clusterName]
	if !ok {
		return nil, false
	}
	ng, ok := ngs[ngName]
	return ng, ok
}

func (s *Store) ListNodegroups(clusterName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ngs, ok := s.nodegroups[clusterName]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(ngs))
	for name := range ngs {
		names = append(names, name)
	}
	return names
}

func (s *Store) UpdateNodegroupConfig(clusterName, ngName string, scaling *NodegroupScalingConfig, labels map[string]string, taints []Taint) (*Nodegroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ngs, ok := s.nodegroups[clusterName]
	if !ok {
		return nil, false
	}
	ng, ok := ngs[ngName]
	if !ok {
		return nil, false
	}
	if scaling != nil {
		ng.ScalingConfig = scaling
	}
	if labels != nil {
		ng.Labels = labels
	}
	if taints != nil {
		ng.Taints = taints
	}
	ng.Status = "UPDATING"
	if ng.Lifecycle != nil {
		ng.Lifecycle.ForceState("UPDATING")
	}
	return ng, true
}

func (s *Store) DeleteNodegroup(clusterName, ngName string) (*Nodegroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ngs, ok := s.nodegroups[clusterName]
	if !ok {
		return nil, false
	}
	ng, ok := ngs[ngName]
	if !ok {
		return nil, false
	}
	ng.Status = "DELETING"
	if ng.Lifecycle != nil {
		ng.Lifecycle.Stop()
	}
	delete(ngs, ngName)
	return ng, true
}

// ---- FargateProfile operations ----

func (s *Store) CreateFargateProfile(clusterName, profileName, podExecRoleARN string, subnetIDs []string, selectors []FargateSelector, tags map[string]string) (*FargateProfile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, false
	}

	profiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		profiles = make(map[string]*FargateProfile)
		s.fargateProfiles[clusterName] = profiles
	}
	if _, exists := profiles[profileName]; exists {
		return nil, false
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 1 * time.Second},
	}
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleCfg)

	fp := &FargateProfile{
		Name:                profileName,
		ARN:                 s.fargateProfileARN(clusterName, profileName),
		ClusterName:         clusterName,
		PodExecutionRoleARN: podExecRoleARN,
		Status:              "CREATING",
		SubnetIDs:           subnetIDs,
		Selectors:           selectors,
		CreatedAt:           time.Now().UTC(),
		Tags:                tags,
		Lifecycle:           lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if profiles, ok := s.fargateProfiles[clusterName]; ok {
			if f, ok := profiles[profileName]; ok {
				f.Status = string(to)
			}
		}
	})

	profiles[profileName] = fp
	return fp, true
}

func (s *Store) GetFargateProfile(clusterName, profileName string) (*FargateProfile, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		return nil, false
	}
	fp, ok := profiles[profileName]
	return fp, ok
}

func (s *Store) ListFargateProfiles(clusterName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(profiles))
	for name := range profiles {
		names = append(names, name)
	}
	return names
}

func (s *Store) DeleteFargateProfile(clusterName, profileName string) (*FargateProfile, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profiles, ok := s.fargateProfiles[clusterName]
	if !ok {
		return nil, false
	}
	fp, ok := profiles[profileName]
	if !ok {
		return nil, false
	}
	fp.Status = "DELETING"
	if fp.Lifecycle != nil {
		fp.Lifecycle.Stop()
	}
	delete(profiles, profileName)
	return fp, true
}

// ---- Addon operations ----

func (s *Store) CreateAddon(clusterName, addonName, addonVersion, serviceAccountRoleARN string, tags map[string]string) (*Addon, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clusters[clusterName]; !ok {
		return nil, false
	}

	addons, ok := s.addons[clusterName]
	if !ok {
		addons = make(map[string]*Addon)
		s.addons[clusterName] = addons
	}
	if _, exists := addons[addonName]; exists {
		return nil, false
	}

	if tags == nil {
		tags = make(map[string]string)
	}

	transitions := []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 1 * time.Second},
	}
	lm := lifecycle.NewMachine("CREATING", transitions, s.lifecycleCfg)

	now := time.Now().UTC()
	addon := &Addon{
		Name:                  addonName,
		ARN:                   s.addonARN(clusterName, addonName),
		ClusterName:           clusterName,
		AddonVersion:          addonVersion,
		Status:                "CREATING",
		ServiceAccountRoleARN: serviceAccountRoleARN,
		CreatedAt:             now,
		ModifiedAt:            now,
		Tags:                  tags,
		Lifecycle:             lm,
	}

	lm.OnTransition(func(from, to lifecycle.State) {
		s.mu.Lock()
		defer s.mu.Unlock()
		if addons, ok := s.addons[clusterName]; ok {
			if a, ok := addons[addonName]; ok {
				a.Status = string(to)
			}
		}
	})

	addons[addonName] = addon
	return addon, true
}

func (s *Store) GetAddon(clusterName, addonName string) (*Addon, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addons, ok := s.addons[clusterName]
	if !ok {
		return nil, false
	}
	addon, ok := addons[addonName]
	return addon, ok
}

func (s *Store) ListAddons(clusterName string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	addons, ok := s.addons[clusterName]
	if !ok {
		return nil
	}
	names := make([]string, 0, len(addons))
	for name := range addons {
		names = append(names, name)
	}
	return names
}

func (s *Store) DeleteAddon(clusterName, addonName string) (*Addon, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	addons, ok := s.addons[clusterName]
	if !ok {
		return nil, false
	}
	addon, ok := addons[addonName]
	if !ok {
		return nil, false
	}
	addon.Status = "DELETING"
	if addon.Lifecycle != nil {
		addon.Lifecycle.Stop()
	}
	delete(addons, addonName)
	return addon, true
}

// ---- Tag operations ----

func (s *Store) TagResource(arn string, tags map[string]string) bool {
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

func (s *Store) UntagResource(arn string, keys []string) bool {
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
	for _, c := range s.clusters {
		if c.ARN == arn {
			return c.Tags
		}
	}
	for _, ngs := range s.nodegroups {
		for _, ng := range ngs {
			if ng.ARN == arn {
				return ng.Tags
			}
		}
	}
	for _, fps := range s.fargateProfiles {
		for _, fp := range fps {
			if fp.ARN == arn {
				return fp.Tags
			}
		}
	}
	for _, addons := range s.addons {
		for _, a := range addons {
			if a.ARN == arn {
				return a.Tags
			}
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

func newID() string {
	return randomHex(12)
}
