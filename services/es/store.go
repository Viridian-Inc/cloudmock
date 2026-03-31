package es

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Domain represents an Elasticsearch domain.
type Domain struct {
	DomainName          string
	ARN                 string
	DomainId            string
	ElasticsearchVersion string
	Endpoint            string
	Processing          bool
	Created             bool
	Deleted             bool
	InstanceType        string
	InstanceCount       int
	Tags                map[string]string
	CreatedTime         time.Time
	Lifecycle           *lifecycle.Machine
}

// Store manages all Elasticsearch resources.
type Store struct {
	mu        sync.RWMutex
	domains   map[string]*Domain
	accountID string
	region    string
	lcConfig  *lifecycle.Config
}

// NewStore creates a new Elasticsearch store.
func NewStore(accountID, region string) *Store {
	return &Store{
		domains:   make(map[string]*Domain),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
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

func (s *Store) domainARN(name string) string {
	return fmt.Sprintf("arn:aws:es:%s:%s:domain/%s", s.region, s.accountID, name)
}

func domainTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "Processing", To: "Active", Delay: 5 * time.Second},
	}
}

func (s *Store) CreateDomain(name, version, instanceType string, instanceCount int, tags map[string]string) (*Domain, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[name]; ok {
		return nil, false
	}
	if version == "" {
		version = "7.10"
	}
	if instanceType == "" {
		instanceType = "r6g.large.elasticsearch"
	}
	if instanceCount == 0 {
		instanceCount = 1
	}
	d := &Domain{
		DomainName:          name,
		ARN:                 s.domainARN(name),
		DomainId:            randomHex(16),
		ElasticsearchVersion: version,
		Endpoint:            fmt.Sprintf("search-%s-%s.%s.es.amazonaws.com", name, randomHex(8), s.region),
		Processing:          true,
		Created:             true,
		InstanceType:        instanceType,
		InstanceCount:       instanceCount,
		Tags:                tags,
		CreatedTime:         time.Now().UTC(),
		Lifecycle:           lifecycle.NewMachine("Processing", domainTransitions(), s.lcConfig),
	}
	s.domains[name] = d
	return d, true
}

func (s *Store) GetDomain(name string) (*Domain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[name]
	if ok {
		d.Processing = string(d.Lifecycle.State()) == "Processing"
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

func (s *Store) UpdateDomainConfig(name, version, instanceType string, instanceCount int) (*Domain, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.domains[name]
	if !ok {
		return nil, false
	}
	if version != "" {
		d.ElasticsearchVersion = version
	}
	if instanceType != "" {
		d.InstanceType = instanceType
	}
	if instanceCount > 0 {
		d.InstanceCount = instanceCount
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
