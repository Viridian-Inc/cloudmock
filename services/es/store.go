package es

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// ESDocument represents an indexed document.
type ESDocument struct {
	ID     string
	Index  string
	Source map[string]any
}

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
	documents map[string]map[string][]ESDocument // domainName -> index -> documents
	accountID string
	region    string
	lcConfig  *lifecycle.Config
}

// NewStore creates a new Elasticsearch store.
func NewStore(accountID, region string) *Store {
	return &Store{
		domains:   make(map[string]*Domain),
		documents: make(map[string]map[string][]ESDocument),
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
		Endpoint:            fmt.Sprintf("%s.%s.es.amazonaws.com", name, s.region),
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

// ---- Document operations ----

func (s *Store) IndexDocument(domainName, index, docID string, source map[string]any) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.domains[domainName]; !ok {
		return "", false
	}
	if s.documents[domainName] == nil {
		s.documents[domainName] = make(map[string][]ESDocument)
	}
	if docID == "" {
		docID = randomHex(10)
	}
	docs := s.documents[domainName][index]
	for i, d := range docs {
		if d.ID == docID {
			s.documents[domainName][index][i].Source = source
			return docID, true
		}
	}
	s.documents[domainName][index] = append(docs, ESDocument{ID: docID, Index: index, Source: source})
	return docID, true
}

func (s *Store) SearchDocuments(domainName, index string, query map[string]any) ([]ESDocument, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.domains[domainName]; !ok {
		return nil, false
	}
	docs := s.documents[domainName][index]
	if query == nil || len(query) == 0 {
		result := make([]ESDocument, len(docs))
		copy(result, docs)
		return result, true
	}
	matchQuery, hasMatch := query["match"].(map[string]any)
	if !hasMatch {
		result := make([]ESDocument, len(docs))
		copy(result, docs)
		return result, true
	}
	var result []ESDocument
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

func (s *Store) ClusterHealth(domainName string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	d, ok := s.domains[domainName]
	if !ok {
		return "", false
	}
	processing := string(d.Lifecycle.State()) == "Processing"
	if processing || d.Deleted {
		return "red", true
	}
	if d.InstanceCount >= 2 {
		return "green", true
	}
	return "yellow", true
}
