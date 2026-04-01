package s3tables

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sync"
	"time"
)

var bucketNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9\-]{0,61}[a-z0-9]?$`)

// TableBucket represents an S3 Tables table bucket.
type TableBucket struct {
	TableBucketARN string
	Name           string
	OwnerAccountID string
	CreatedAt      time.Time
}

// Table represents an S3 Tables table.
type Table struct {
	TableARN       string
	Namespace      string
	Name           string
	TableBucketARN string
	Format         string
	Type           string
	CreatedAt      time.Time
	ModifiedAt     time.Time
}

// TablePolicy represents a policy for an S3 Tables table.
type TablePolicy struct {
	TableARN       string
	ResourcePolicy string
}

// Store manages S3 Tables resources in memory.
type Store struct {
	mu            sync.RWMutex
	tableBuckets  map[string]*TableBucket // name -> bucket
	tables        map[string]map[string]*Table // bucketARN -> tableName -> table
	policies      map[string]*TablePolicy // tableARN -> policy
	accountID     string
	region        string
}

// NewStore returns a new empty S3 Tables Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		tableBuckets: make(map[string]*TableBucket),
		tables:       make(map[string]map[string]*Table),
		policies:     make(map[string]*TablePolicy),
		accountID:    accountID,
		region:       region,
	}
}

// CreateTableBucket creates a new table bucket.
func (s *Store) CreateTableBucket(name string) (*TableBucket, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !bucketNameRegex.MatchString(name) {
		return nil, fmt.Errorf("invalid bucket name: must be lowercase, 3-63 chars, no underscores: %s", name)
	}

	if _, ok := s.tableBuckets[name]; ok {
		return nil, fmt.Errorf("table bucket already exists: %s", name)
	}

	arn := fmt.Sprintf("arn:aws:s3tables:%s:%s:bucket/%s", s.region, s.accountID, name)
	bucket := &TableBucket{
		TableBucketARN: arn,
		Name:           name,
		OwnerAccountID: s.accountID,
		CreatedAt:      time.Now().UTC(),
	}
	s.tableBuckets[name] = bucket
	s.tables[arn] = make(map[string]*Table)
	return bucket, nil
}

// GetTableBucket retrieves a table bucket by ARN.
func (s *Store) GetTableBucket(arn string) (*TableBucket, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, b := range s.tableBuckets {
		if b.TableBucketARN == arn {
			return b, true
		}
	}
	return nil, false
}

// ListTableBuckets returns all table buckets.
func (s *Store) ListTableBuckets() []*TableBucket {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*TableBucket, 0, len(s.tableBuckets))
	for _, b := range s.tableBuckets {
		out = append(out, b)
	}
	return out
}

// DeleteTableBucket removes a table bucket.
func (s *Store) DeleteTableBucket(arn string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for name, b := range s.tableBuckets {
		if b.TableBucketARN == arn {
			delete(s.tableBuckets, name)
			delete(s.tables, arn)
			return true
		}
	}
	return false
}

// CreateTable creates a new table in a bucket.
func (s *Store) CreateTable(bucketARN, namespace, name, format string) (*Table, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if namespace == "" {
		return nil, fmt.Errorf("namespace is required")
	}
	if name == "" {
		return nil, fmt.Errorf("table name is required")
	}

	tableMap, ok := s.tables[bucketARN]
	if !ok {
		return nil, fmt.Errorf("table bucket not found: %s", bucketARN)
	}

	key := namespace + "/" + name
	if _, ok := tableMap[key]; ok {
		return nil, fmt.Errorf("table already exists: %s/%s", namespace, name)
	}

	now := time.Now().UTC()
	tableARN := fmt.Sprintf("%s/table/%s/%s", bucketARN, namespace, name)
	table := &Table{
		TableARN:       tableARN,
		Namespace:      namespace,
		Name:           name,
		TableBucketARN: bucketARN,
		Format:         format,
		Type:           "customer",
		CreatedAt:      now,
		ModifiedAt:     now,
	}
	tableMap[key] = table
	return table, nil
}

// GetTable retrieves a table.
func (s *Store) GetTable(bucketARN, namespace, name string) (*Table, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tableMap, ok := s.tables[bucketARN]
	if !ok {
		return nil, false
	}
	t, ok := tableMap[namespace+"/"+name]
	return t, ok
}

// ListTables returns all tables in a bucket.
func (s *Store) ListTables(bucketARN string) []*Table {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tableMap := s.tables[bucketARN]
	out := make([]*Table, 0, len(tableMap))
	for _, t := range tableMap {
		out = append(out, t)
	}
	return out
}

// DeleteTable removes a table.
func (s *Store) DeleteTable(bucketARN, namespace, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tableMap, ok := s.tables[bucketARN]
	if !ok {
		return false
	}
	key := namespace + "/" + name
	if _, ok := tableMap[key]; !ok {
		return false
	}
	tableARN := tableMap[key].TableARN
	delete(tableMap, key)
	delete(s.policies, tableARN)
	return true
}

// PutTablePolicy sets a policy for a table.
func (s *Store) PutTablePolicy(tableARN, policy string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate JSON structure
	var parsed map[string]any
	if err := json.Unmarshal([]byte(policy), &parsed); err != nil {
		return fmt.Errorf("policy is not valid JSON: %v", err)
	}

	s.policies[tableARN] = &TablePolicy{
		TableARN:       tableARN,
		ResourcePolicy: policy,
	}
	return nil
}

// GetTablePolicy retrieves a table policy.
func (s *Store) GetTablePolicy(tableARN string) (*TablePolicy, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[tableARN]
	return p, ok
}

// DeleteTablePolicy removes a table policy.
func (s *Store) DeleteTablePolicy(tableARN string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.policies[tableARN]; !ok {
		return false
	}
	delete(s.policies, tableARN)
	return true
}
