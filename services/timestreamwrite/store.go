package timestreamwrite

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
)

// Database represents a Timestream database.
type Database struct {
	Name         string
	ARN          string
	Status       string
	KmsKeyId     string
	TableCount   int
	CreationTime time.Time
	LastUpdatedTime time.Time
	Tags         map[string]string
	Lifecycle    *lifecycle.Machine
}

// Table represents a Timestream table.
type Table struct {
	Name             string
	ARN              string
	DatabaseName     string
	Status           string
	RetentionProperties RetentionProperties
	MagneticStoreWriteProperties MagneticStoreWriteProperties
	CreationTime     time.Time
	LastUpdatedTime  time.Time
	Tags             map[string]string
	Lifecycle        *lifecycle.Machine
}

// RetentionProperties holds data retention config.
type RetentionProperties struct {
	MemoryStoreRetentionPeriodInHours  int64
	MagneticStoreRetentionPeriodInDays int64
}

// MagneticStoreWriteProperties holds magnetic store config.
type MagneticStoreWriteProperties struct {
	EnableMagneticStoreWrites bool
}

// Record represents a Timestream record (simplified).
type Record struct {
	Dimensions       []Dimension
	MeasureName      string
	MeasureValue     string
	MeasureValueType string
	Time             string
	TimeUnit         string
}

// Dimension represents a record dimension.
type Dimension struct {
	Name  string
	Value string
}

// Store manages all Timestream resources.
type Store struct {
	mu        sync.RWMutex
	databases map[string]*Database
	tables    map[string]map[string]*Table // dbName -> tableName -> table
	records   map[string]map[string][]Record // dbName -> tableName -> records
	accountID string
	region    string
	lcConfig  *lifecycle.Config
}

// NewStore creates a new Timestream store.
func NewStore(accountID, region string) *Store {
	return &Store{
		databases: make(map[string]*Database),
		tables:    make(map[string]map[string]*Table),
		records:   make(map[string]map[string][]Record),
		accountID: accountID,
		region:    region,
		lcConfig:  lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) databaseARN(name string) string {
	return fmt.Sprintf("arn:aws:timestream:%s:%s:database/%s", s.region, s.accountID, name)
}

func (s *Store) tableARN(dbName, tableName string) string {
	return fmt.Sprintf("arn:aws:timestream:%s:%s:database/%s/table/%s", s.region, s.accountID, dbName, tableName)
}

func dbTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 2 * time.Second},
		{From: "DELETING", To: "DELETED", Delay: 2 * time.Second},
	}
}

func tableTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "CREATING", To: "ACTIVE", Delay: 2 * time.Second},
		{From: "DELETING", To: "DELETED", Delay: 2 * time.Second},
	}
}

// ---- Database operations ----

func (s *Store) CreateDatabase(name, kmsKeyId string, tags map[string]string) (*Database, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[name]; ok {
		return nil, false
	}
	now := time.Now().UTC()
	db := &Database{
		Name:            name,
		ARN:             s.databaseARN(name),
		Status:          "ACTIVE",
		KmsKeyId:        kmsKeyId,
		CreationTime:    now,
		LastUpdatedTime: now,
		Tags:            tags,
		Lifecycle:       lifecycle.NewMachine("CREATING", dbTransitions(), s.lcConfig),
	}
	s.databases[name] = db
	s.tables[name] = make(map[string]*Table)
	s.records[name] = make(map[string][]Record)
	return db, true
}

func (s *Store) GetDatabase(name string) (*Database, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db, ok := s.databases[name]
	if ok {
		db.Status = string(db.Lifecycle.State())
		db.TableCount = len(s.tables[name])
	}
	return db, ok
}

func (s *Store) ListDatabases() []*Database {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Database, 0, len(s.databases))
	for _, db := range s.databases {
		db.Status = string(db.Lifecycle.State())
		db.TableCount = len(s.tables[db.Name])
		result = append(result, db)
	}
	return result
}

func (s *Store) UpdateDatabase(name, kmsKeyId string) (*Database, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, ok := s.databases[name]
	if !ok {
		return nil, false
	}
	if kmsKeyId != "" {
		db.KmsKeyId = kmsKeyId
	}
	db.LastUpdatedTime = time.Now().UTC()
	return db, true
}

func (s *Store) DeleteDatabase(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	db, ok := s.databases[name]
	if !ok {
		return false
	}
	db.Lifecycle.Stop()
	delete(s.databases, name)
	delete(s.tables, name)
	delete(s.records, name)
	return true
}

// ---- Table operations ----

func (s *Store) CreateTable(dbName, tableName string, retention RetentionProperties, magnetic MagneticStoreWriteProperties, tags map[string]string) (*Table, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[dbName]; !ok {
		return nil, false
	}
	tbls := s.tables[dbName]
	if _, exists := tbls[tableName]; exists {
		return nil, false
	}
	if retention.MemoryStoreRetentionPeriodInHours == 0 {
		retention.MemoryStoreRetentionPeriodInHours = 6
	}
	if retention.MagneticStoreRetentionPeriodInDays == 0 {
		retention.MagneticStoreRetentionPeriodInDays = 73000
	}
	now := time.Now().UTC()
	t := &Table{
		Name:                         tableName,
		ARN:                          s.tableARN(dbName, tableName),
		DatabaseName:                 dbName,
		Status:                       "ACTIVE",
		RetentionProperties:          retention,
		MagneticStoreWriteProperties: magnetic,
		CreationTime:                 now,
		LastUpdatedTime:              now,
		Tags:                         tags,
		Lifecycle:                    lifecycle.NewMachine("CREATING", tableTransitions(), s.lcConfig),
	}
	tbls[tableName] = t
	s.records[dbName][tableName] = make([]Record, 0)
	return t, true
}

func (s *Store) GetTable(dbName, tableName string) (*Table, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tbls := s.tables[dbName]
	if tbls == nil {
		return nil, false
	}
	t, ok := tbls[tableName]
	if ok {
		t.Status = string(t.Lifecycle.State())
	}
	return t, ok
}

func (s *Store) ListTables(dbName string) []*Table {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tbls := s.tables[dbName]
	result := make([]*Table, 0, len(tbls))
	for _, t := range tbls {
		t.Status = string(t.Lifecycle.State())
		result = append(result, t)
	}
	return result
}

func (s *Store) UpdateTable(dbName, tableName string, retention *RetentionProperties, magnetic *MagneticStoreWriteProperties) (*Table, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	tbls := s.tables[dbName]
	if tbls == nil {
		return nil, false
	}
	t, ok := tbls[tableName]
	if !ok {
		return nil, false
	}
	if retention != nil {
		t.RetentionProperties = *retention
	}
	if magnetic != nil {
		t.MagneticStoreWriteProperties = *magnetic
	}
	t.LastUpdatedTime = time.Now().UTC()
	return t, true
}

func (s *Store) DeleteTable(dbName, tableName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tbls := s.tables[dbName]
	if tbls == nil {
		return false
	}
	t, ok := tbls[tableName]
	if !ok {
		return false
	}
	t.Lifecycle.Stop()
	delete(tbls, tableName)
	delete(s.records[dbName], tableName)
	return true
}

// ---- Record operations ----

func (s *Store) WriteRecords(dbName, tableName string, records []Record) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.records[dbName] == nil {
		return false
	}
	if _, ok := s.tables[dbName][tableName]; !ok {
		return false
	}
	s.records[dbName][tableName] = append(s.records[dbName][tableName], records...)
	return true
}

// ---- Tag operations ----

func (s *Store) tagMapByARN(arn string) map[string]string {
	for _, db := range s.databases {
		if db.ARN == arn {
			return db.Tags
		}
	}
	for _, tbls := range s.tables {
		for _, t := range tbls {
			if t.ARN == arn {
				return t.Tags
			}
		}
	}
	return nil
}

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
