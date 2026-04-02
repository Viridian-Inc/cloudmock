package opensearch

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Document represents an indexed document.
type Document struct {
	ID     string
	Index  string
	Source map[string]any
}

// Domain represents an OpenSearch domain.
type Domain struct {
	DomainName     string
	ARN            string
	DomainId       string
	EngineVersion  string
	Endpoint       string
	Processing     bool
	Created        bool
	Deleted        bool
	ClusterConfig  ClusterConfig
	EBSOptions     EBSOptions
	Tags           map[string]string
	CreatedTime    time.Time
	Lifecycle      *lifecycle.Machine
}

// ClusterConfig holds cluster configuration.
type ClusterConfig struct {
	InstanceType          string
	InstanceCount         int
	DedicatedMasterEnabled bool
	DedicatedMasterType   string
	DedicatedMasterCount  int
}

// EBSOptions holds EBS volume configuration.
type EBSOptions struct {
	EBSEnabled bool
	VolumeType string
	VolumeSize int
}

// UpgradeStatus tracks a domain upgrade.
type UpgradeStatus struct {
	StepStatus    string
	UpgradeName   string
	UpgradeStep   string
}

// VpcEndpoint represents an OpenSearch VPC endpoint.
type VpcEndpoint struct {
	VpcEndpointID   string
	DomainArn       string
	VpcOptions      VpcOptions
	Status          string
	Endpoint        string
}

// VpcOptions holds VPC configuration for a domain or endpoint.
type VpcOptions struct {
	VpcID            string
	SubnetIDs        []string
	SecurityGroupIDs []string
}

// CompatibleVersion holds a source engine version and compatible targets.
type CompatibleVersion struct {
	SourceVersion   string
	TargetVersions  []string
}

// Store manages all OpenSearch resources.
type Store struct {
	mu           sync.RWMutex
	domains      map[string]*Domain
	upgrades     map[string]*UpgradeStatus
	documents    map[string]map[string][]Document // domainName -> index -> documents
	vpcEndpoints map[string]*VpcEndpoint          // vpcEndpointID -> endpoint
	accountID    string
	region       string
	lcConfig     *lifecycle.Config
}

// NewStore creates a new OpenSearch store.
func NewStore(accountID, region string) *Store {
	return &Store{
		domains:      make(map[string]*Domain),
		upgrades:     make(map[string]*UpgradeStatus),
		documents:    make(map[string]map[string][]Document),
		vpcEndpoints: make(map[string]*VpcEndpoint),
		accountID:    accountID,
		region:       region,
		lcConfig:     lifecycle.DefaultConfig(),
	}
}

func randomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (s *Store) domainARN(name string) string {
	return fmt.Sprintf("arn:aws:es:%s:%s:domain/%s", s.region, s.accountID, name)
}

func domainTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "Processing", To: "Active", Delay: 5 * time.Second},
	}
}

func (s *Store) CreateDomain(name, engineVersion string, clusterConfig ClusterConfig, ebsOptions EBSOptions, tags map[string]string) (*Domain, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[name]; ok {
		return nil, false
	}
	if engineVersion == "" {
		engineVersion = "OpenSearch_2.11"
	}
	if clusterConfig.InstanceType == "" {
		clusterConfig.InstanceType = "r6g.large.search"
	}
	if clusterConfig.InstanceCount == 0 {
		clusterConfig.InstanceCount = 1
	}
	d := &Domain{
		DomainName:    name,
		ARN:           s.domainARN(name),
		DomainId:      randomHex(16),
		EngineVersion: engineVersion,
		Endpoint:      fmt.Sprintf("%s.%s.es.amazonaws.com", name, s.region),
		Processing:    true,
		Created:       true,
		ClusterConfig: clusterConfig,
		EBSOptions:    ebsOptions,
		Tags:          tags,
		CreatedTime:   time.Now().UTC(),
		Lifecycle:     lifecycle.NewMachine("Processing", domainTransitions(), s.lcConfig),
	}
	s.domains[name] = d
	return d, true
}

func (s *Store) GetDomain(name string) (*Domain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[name]
	if ok {
		state := string(d.Lifecycle.State())
		d.Processing = state == "Processing"
	}
	return d, ok
}

func (s *Store) ListDomainNames() []*Domain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Domain, 0, len(s.domains))
	for _, d := range s.domains {
		if !d.Deleted {
			result = append(result, d)
		}
	}
	return result
}

func (s *Store) DeleteDomain(name string) (*Domain, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.domains[name]
	if !ok {
		return nil, false
	}
	d.Deleted = true
	d.Lifecycle.Stop()
	delete(s.domains, name)
	return d, true
}

func (s *Store) UpdateDomainConfig(name, engineVersion string, clusterConfig *ClusterConfig, ebsOptions *EBSOptions) (*Domain, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.domains[name]
	if !ok {
		return nil, false
	}
	if engineVersion != "" {
		d.EngineVersion = engineVersion
	}
	if clusterConfig != nil {
		d.ClusterConfig = *clusterConfig
	}
	if ebsOptions != nil {
		d.EBSOptions = *ebsOptions
	}
	return d, true
}

func (s *Store) AddTags(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			for k, v := range tags {
				d.Tags[k] = v
			}
			return true
		}
	}
	return false
}

func (s *Store) RemoveTags(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			for _, k := range keys {
				delete(d.Tags, k)
			}
			return true
		}
	}
	return false
}

func (s *Store) ListTags(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, d := range s.domains {
		if d.ARN == arn {
			result := make(map[string]string, len(d.Tags))
			for k, v := range d.Tags {
				result[k] = v
			}
			return result, true
		}
	}
	return nil, false
}

func (s *Store) UpgradeDomain(name, targetVersion string) (*UpgradeStatus, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.domains[name]
	if !ok {
		return nil, false
	}
	status := &UpgradeStatus{
		StepStatus:  "SUCCEEDED",
		UpgradeName: targetVersion,
		UpgradeStep: "UPGRADE",
	}
	d.EngineVersion = targetVersion
	s.upgrades[name] = status
	return status, true
}

func (s *Store) GetUpgradeStatus(name string) (*UpgradeStatus, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	status, ok := s.upgrades[name]
	return status, ok
}

// ---- Document operations ----

func (s *Store) IndexDocument(domainName, index, docID string, source map[string]any) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[domainName]; !ok {
		return "", false
	}
	if s.documents[domainName] == nil {
		s.documents[domainName] = make(map[string][]Document)
	}
	if docID == "" {
		docID = randomHex(10)
	}
	// Check if document already exists (update)
	docs := s.documents[domainName][index]
	for i, d := range docs {
		if d.ID == docID {
			s.documents[domainName][index][i].Source = source
			return docID, true
		}
	}
	s.documents[domainName][index] = append(docs, Document{
		ID:     docID,
		Index:  index,
		Source: source,
	})
	return docID, true
}

func (s *Store) SearchDocuments(domainName, index string, query map[string]any) ([]Document, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.domains[domainName]; !ok {
		return nil, false
	}
	docs := s.documents[domainName][index]
	if query == nil || len(query) == 0 {
		// Return all documents
		result := make([]Document, len(docs))
		copy(result, docs)
		return result, true
	}

	// Simple equality matching on "match" queries
	matchQuery, hasMatch := query["match"].(map[string]any)
	if !hasMatch {
		// No match filter — return all
		result := make([]Document, len(docs))
		copy(result, docs)
		return result, true
	}

	var result []Document
	for _, doc := range docs {
		matches := true
		for field, value := range matchQuery {
			docVal, exists := doc.Source[field]
			if !exists || fmt.Sprintf("%v", docVal) != fmt.Sprintf("%v", value) {
				matches = false
				break
			}
		}
		if matches {
			result = append(result, doc)
		}
	}
	return result, true
}

// DescribeDomains returns multiple domains by name.
func (s *Store) DescribeDomains(names []string) []*Domain {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Domain, 0, len(names))
	for _, name := range names {
		if d, ok := s.domains[name]; ok {
			state := string(d.Lifecycle.State())
			d.Processing = state == "Processing"
			result = append(result, d)
		}
	}
	return result
}

// GetCompatibleVersions returns version compatibility info.
func (s *Store) GetCompatibleVersions(domainName string) []CompatibleVersion {
	compatMap := map[string][]string{
		"OpenSearch_1.0":  {"OpenSearch_1.1", "OpenSearch_1.2", "OpenSearch_1.3"},
		"OpenSearch_1.1":  {"OpenSearch_1.2", "OpenSearch_1.3"},
		"OpenSearch_1.2":  {"OpenSearch_1.3", "OpenSearch_2.0"},
		"OpenSearch_1.3":  {"OpenSearch_2.0", "OpenSearch_2.3"},
		"OpenSearch_2.0":  {"OpenSearch_2.3", "OpenSearch_2.5"},
		"OpenSearch_2.3":  {"OpenSearch_2.5", "OpenSearch_2.7"},
		"OpenSearch_2.5":  {"OpenSearch_2.7", "OpenSearch_2.9"},
		"OpenSearch_2.7":  {"OpenSearch_2.9", "OpenSearch_2.11"},
		"OpenSearch_2.9":  {"OpenSearch_2.11"},
		"OpenSearch_2.11": {},
		"Elasticsearch_7.1":  {"Elasticsearch_7.4", "Elasticsearch_7.7"},
		"Elasticsearch_7.4":  {"Elasticsearch_7.7", "Elasticsearch_7.10"},
		"Elasticsearch_7.7":  {"Elasticsearch_7.10", "OpenSearch_1.0"},
		"Elasticsearch_7.10": {"OpenSearch_1.0", "OpenSearch_1.1"},
	}

	if domainName != "" {
		s.mu.RLock()
		d, ok := s.domains[domainName]
		s.mu.RUnlock()
		if ok {
			targets := compatMap[d.EngineVersion]
			if targets == nil {
				targets = []string{}
			}
			return []CompatibleVersion{{SourceVersion: d.EngineVersion, TargetVersions: targets}}
		}
	}

	result := make([]CompatibleVersion, 0, len(compatMap))
	for src, tgts := range compatMap {
		result = append(result, CompatibleVersion{SourceVersion: src, TargetVersions: tgts})
	}
	return result
}

// ---- VPC Endpoint operations ----

func (s *Store) vpcEndpointARN(id string) string {
	return fmt.Sprintf("arn:aws:es:%s:%s:vpcendpoint/%s", s.region, s.accountID, id)
}

func (s *Store) CreateVpcEndpoint(domainArn string, vpcOptions VpcOptions) (*VpcEndpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := "aos-endpoint-" + randomHex(8)
	ep := &VpcEndpoint{
		VpcEndpointID: id,
		DomainArn:     domainArn,
		VpcOptions:    vpcOptions,
		Status:        "ACTIVE",
		Endpoint:      fmt.Sprintf("vpc-endpoint-%s.%s.es.amazonaws.com", randomHex(8), s.region),
	}
	s.vpcEndpoints[id] = ep
	return ep, true
}

func (s *Store) DescribeVpcEndpoints(ids []string) []*VpcEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(ids) == 0 {
		result := make([]*VpcEndpoint, 0, len(s.vpcEndpoints))
		for _, ep := range s.vpcEndpoints {
			result = append(result, ep)
		}
		return result
	}
	result := make([]*VpcEndpoint, 0, len(ids))
	for _, id := range ids {
		if ep, ok := s.vpcEndpoints[id]; ok {
			result = append(result, ep)
		}
	}
	return result
}

func (s *Store) ListVpcEndpoints(domainArn string) []*VpcEndpoint {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*VpcEndpoint, 0)
	for _, ep := range s.vpcEndpoints {
		if domainArn == "" || ep.DomainArn == domainArn {
			result = append(result, ep)
		}
	}
	return result
}

func (s *Store) DeleteVpcEndpoint(id string) (*VpcEndpoint, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ep, ok := s.vpcEndpoints[id]
	if !ok {
		return nil, false
	}
	delete(s.vpcEndpoints, id)
	return ep, true
}

// ClusterHealth returns health status based on replica configuration.
func (s *Store) ClusterHealth(domainName string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[domainName]
	if !ok {
		return "", false
	}
	// Update processing state from lifecycle
	processing := string(d.Lifecycle.State()) == "Processing"
	if processing || d.Deleted {
		return "red", true
	}
	if d.ClusterConfig.InstanceCount >= 2 {
		return "green", true
	}
	return "yellow", true
}
