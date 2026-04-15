package keyspaces

import (
	"fmt"
	"sync"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

// StoredKeyspace is a persisted keyspace.
type StoredKeyspace struct {
	Name              string
	Arn               string
	ReplicationType   string
	ReplicationRegions []string
	Tags              map[string]string
	CreatedAt         time.Time
}

// StoredTable is a persisted table.
type StoredTable struct {
	KeyspaceName    string
	TableName       string
	Arn             string
	Status          string
	SchemaDefinition map[string]any
	CapacitySpec    map[string]any
	EncryptionSpec  map[string]any
	PITR            map[string]any
	TTL             map[string]any
	ClientEncoding  string
	DefaultTimeToLive int
	Comment         map[string]any
	CreatedAt       time.Time
	Tags            map[string]string
}

// StoredType is a persisted user-defined type.
type StoredType struct {
	KeyspaceName string
	TypeName     string
	Arn          string
	Fields       []map[string]any
	DirectReferringTables []string
	DirectParentTypes     []string
	MaxNestingDepth       int
	KeyspaceArn           string
	CreatedAt             time.Time
}

// Store is the in-memory data store for keyspaces resources.
type Store struct {
	mu        sync.RWMutex
	accountID string
	region    string
	keyspaces map[string]*StoredKeyspace            // name -> keyspace
	tables    map[string]map[string]*StoredTable    // keyspace -> table name -> table
	types     map[string]map[string]*StoredType     // keyspace -> type name -> type
	tags      map[string]map[string]string          // arn -> tags
}

// NewStore creates an empty Store.
func NewStore(accountID, region string) *Store {
	return &Store{
		accountID: accountID,
		region:    region,
		keyspaces: make(map[string]*StoredKeyspace),
		tables:    make(map[string]map[string]*StoredTable),
		types:     make(map[string]map[string]*StoredType),
		tags:      make(map[string]map[string]string),
	}
}

// Reset clears all in-memory state.
func (s *Store) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keyspaces = make(map[string]*StoredKeyspace)
	s.tables = make(map[string]map[string]*StoredTable)
	s.types = make(map[string]map[string]*StoredType)
	s.tags = make(map[string]map[string]string)
}

// ── Keyspaces ────────────────────────────────────────────────────────────────

func (s *Store) CreateKeyspace(name, replicationType string, replicationRegions []string, tags map[string]string) (*StoredKeyspace, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyspaces[name]; ok {
		return nil, service.NewAWSError("ConflictException",
			"Keyspace already exists: "+name, 409)
	}
	arn := fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s", s.region, s.accountID, name)
	ks := &StoredKeyspace{
		Name:              name,
		Arn:               arn,
		ReplicationType:   replicationType,
		ReplicationRegions: replicationRegions,
		Tags:              copyMap(tags),
		CreatedAt:         time.Now().UTC(),
	}
	s.keyspaces[name] = ks
	return ks, nil
}

func (s *Store) GetKeyspace(name string) (*StoredKeyspace, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ks, ok := s.keyspaces[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Keyspace not found: "+name, 404)
	}
	return ks, nil
}

func (s *Store) DeleteKeyspace(name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyspaces[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Keyspace not found: "+name, 404)
	}
	delete(s.keyspaces, name)
	delete(s.tables, name)
	delete(s.types, name)
	return nil
}

func (s *Store) ListKeyspaces() []*StoredKeyspace {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*StoredKeyspace, 0, len(s.keyspaces))
	for _, ks := range s.keyspaces {
		out = append(out, ks)
	}
	return out
}

func (s *Store) UpdateKeyspace(name string, replicationRegions []string) (*StoredKeyspace, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ks, ok := s.keyspaces[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Keyspace not found: "+name, 404)
	}
	if replicationRegions != nil {
		ks.ReplicationRegions = replicationRegions
	}
	return ks, nil
}

// ── Tables ───────────────────────────────────────────────────────────────────

func (s *Store) CreateTable(keyspace, name string, schema, capacity, encryption, pitr, ttl map[string]any, defaultTTL int, comment map[string]any, tags map[string]string) (*StoredTable, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyspaces[keyspace]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Keyspace not found: "+keyspace, 404)
	}
	if s.tables[keyspace] == nil {
		s.tables[keyspace] = make(map[string]*StoredTable)
	}
	if _, ok := s.tables[keyspace][name]; ok {
		return nil, service.NewAWSError("ConflictException",
			"Table already exists: "+name, 409)
	}
	arn := fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s/table/%s", s.region, s.accountID, keyspace, name)
	t := &StoredTable{
		KeyspaceName:      keyspace,
		TableName:         name,
		Arn:               arn,
		Status:            "ACTIVE",
		SchemaDefinition:  schema,
		CapacitySpec:      capacity,
		EncryptionSpec:    encryption,
		PITR:              pitr,
		TTL:               ttl,
		DefaultTimeToLive: defaultTTL,
		Comment:           comment,
		CreatedAt:         time.Now().UTC(),
		Tags:              copyMap(tags),
	}
	s.tables[keyspace][name] = t
	return t, nil
}

func (s *Store) GetTable(keyspace, name string) (*StoredTable, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ks, ok := s.tables[keyspace]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	t, ok := ks[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	return t, nil
}

func (s *Store) DeleteTable(keyspace, name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ks, ok := s.tables[keyspace]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	if _, ok := ks[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	delete(ks, name)
	return nil
}

func (s *Store) ListTables(keyspace string) []*StoredTable {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ks, ok := s.tables[keyspace]
	if !ok {
		return nil
	}
	out := make([]*StoredTable, 0, len(ks))
	for _, t := range ks {
		out = append(out, t)
	}
	return out
}

func (s *Store) UpdateTable(keyspace, name string, schema, capacity, pitr map[string]any, defaultTTL *int) (*StoredTable, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ks, ok := s.tables[keyspace]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	t, ok := ks[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Table not found: "+name, 404)
	}
	if schema != nil {
		t.SchemaDefinition = schema
	}
	if capacity != nil {
		t.CapacitySpec = capacity
	}
	if pitr != nil {
		t.PITR = pitr
	}
	if defaultTTL != nil {
		t.DefaultTimeToLive = *defaultTTL
	}
	return t, nil
}

func (s *Store) RestoreTable(sourceKeyspace, sourceName, targetKeyspace, targetName string) (*StoredTable, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	srcKS, ok := s.tables[sourceKeyspace]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Source table not found: "+sourceName, 404)
	}
	src, ok := srcKS[sourceName]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Source table not found: "+sourceName, 404)
	}
	if _, ok := s.keyspaces[targetKeyspace]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Target keyspace not found: "+targetKeyspace, 404)
	}
	if s.tables[targetKeyspace] == nil {
		s.tables[targetKeyspace] = make(map[string]*StoredTable)
	}
	if _, ok := s.tables[targetKeyspace][targetName]; ok {
		return nil, service.NewAWSError("ConflictException",
			"Target table already exists: "+targetName, 409)
	}
	arn := fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s/table/%s", s.region, s.accountID, targetKeyspace, targetName)
	copied := *src
	copied.KeyspaceName = targetKeyspace
	copied.TableName = targetName
	copied.Arn = arn
	copied.CreatedAt = time.Now().UTC()
	s.tables[targetKeyspace][targetName] = &copied
	return &copied, nil
}

// ── Types ────────────────────────────────────────────────────────────────────

func (s *Store) CreateType(keyspace, name string, fields []map[string]any) (*StoredType, *service.AWSError) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.keyspaces[keyspace]; !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Keyspace not found: "+keyspace, 404)
	}
	if s.types[keyspace] == nil {
		s.types[keyspace] = make(map[string]*StoredType)
	}
	if _, ok := s.types[keyspace][name]; ok {
		return nil, service.NewAWSError("ConflictException",
			"Type already exists: "+name, 409)
	}
	ksArn := fmt.Sprintf("arn:aws:cassandra:%s:%s:/keyspace/%s", s.region, s.accountID, keyspace)
	t := &StoredType{
		KeyspaceName: keyspace,
		TypeName:     name,
		Arn:          fmt.Sprintf("%s/type/%s", ksArn, name),
		Fields:       fields,
		KeyspaceArn:  ksArn,
		CreatedAt:    time.Now().UTC(),
	}
	s.types[keyspace][name] = t
	return t, nil
}

func (s *Store) GetType(keyspace, name string) (*StoredType, *service.AWSError) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ks, ok := s.types[keyspace]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Type not found: "+name, 404)
	}
	t, ok := ks[name]
	if !ok {
		return nil, service.NewAWSError("ResourceNotFoundException",
			"Type not found: "+name, 404)
	}
	return t, nil
}

func (s *Store) DeleteType(keyspace, name string) *service.AWSError {
	s.mu.Lock()
	defer s.mu.Unlock()
	ks, ok := s.types[keyspace]
	if !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Type not found: "+name, 404)
	}
	if _, ok := ks[name]; !ok {
		return service.NewAWSError("ResourceNotFoundException",
			"Type not found: "+name, 404)
	}
	delete(ks, name)
	return nil
}

func (s *Store) ListTypes(keyspace string) []*StoredType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ks, ok := s.types[keyspace]
	if !ok {
		return nil
	}
	out := make([]*StoredType, 0, len(ks))
	for _, t := range ks {
		out = append(out, t)
	}
	return out
}

// ── Tags ─────────────────────────────────────────────────────────────────────

func (s *Store) TagResource(arn string, tags map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tags[arn] == nil {
		s.tags[arn] = make(map[string]string)
	}
	for k, v := range tags {
		s.tags[arn][k] = v
	}
}

func (s *Store) UntagResource(arn string, keys []string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.tags[arn]; ok {
		for _, k := range keys {
			delete(m, k)
		}
	}
}

func (s *Store) ListTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string)
	if m, ok := s.tags[arn]; ok {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

func copyMap(m map[string]string) map[string]string {
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
