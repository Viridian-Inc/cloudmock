package glue

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/lifecycle"
	"github.com/neureaux/cloudmock/pkg/mocklog"
	"github.com/neureaux/cloudmock/pkg/service"
)

// Database represents a Glue catalog database.
type Database struct {
	Name        string
	Description string
	LocationURI string
	CreateTime  time.Time
	Parameters  map[string]string
}

// Table represents a Glue catalog table.
type Table struct {
	Name         string
	DatabaseName string
	Description  string
	StorageDesc  StorageDescriptor
	Parameters   map[string]string
	CreateTime   time.Time
	UpdateTime   time.Time
}

// StorageDescriptor holds storage info for a table.
type StorageDescriptor struct {
	Location     string
	InputFormat  string
	OutputFormat string
	Columns      []Column
}

// Column represents a table column.
type Column struct {
	Name    string
	Type    string
	Comment string
}

// Crawler represents a Glue crawler.
type Crawler struct {
	Name         string
	Role         string
	DatabaseName string
	Description  string
	Targets      CrawlerTargets
	Schedule     string
	State        string
	Lifecycle    *lifecycle.Machine
	CreateTime   time.Time
	LastUpdated  time.Time
	Tags         map[string]string
	LastCrawl    *LastCrawlInfo
}

// LastCrawlInfo holds metadata about the most recent crawler run.
type LastCrawlInfo struct {
	Status        string // SUCCEEDED, FAILED
	ErrorMessage  string
	LogGroup      string
	LogStream     string
	MessagePrefix string
	StartTime     time.Time
	TablesCreated int
	TablesUpdated int
}

// ServiceLocator resolves other services by name.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// CrawlerTargets holds crawler target configuration.
type CrawlerTargets struct {
	S3Targets []S3Target
}

// S3Target represents an S3 path target for a crawler.
type S3Target struct {
	Path       string
	Exclusions []string
}

// Job represents a Glue ETL job.
type Job struct {
	Name            string
	Role            string
	Command         JobCommand
	Description     string
	MaxRetries      int
	MaxCapacity     float64
	GlueVersion     string
	NumberOfWorkers int
	WorkerType      string
	CreateTime      time.Time
	Tags            map[string]string
}

// JobCommand holds the script location and language.
type JobCommand struct {
	Name           string
	ScriptLocation string
	PythonVersion  string
}

// JobRun represents a single execution of a job.
type JobRun struct {
	ID            string
	JobName       string
	State         string
	StartedOn     time.Time
	CompletedOn   *time.Time
	Lifecycle     *lifecycle.Machine
	ErrorMessage  string
	Attempt       int
	ExecutionTime int
}

// Connection represents a Glue connection.
type Connection struct {
	Name               string
	Description        string
	ConnectionType     string
	ConnectionProperties map[string]string
	PhysicalConnectionRequirements *PhysicalConnectionRequirements
	CreateTime         time.Time
}

// PhysicalConnectionRequirements holds network config.
type PhysicalConnectionRequirements struct {
	SubnetID               string
	SecurityGroupIDList    []string
	AvailabilityZone       string
}

// Store manages all Glue resources in memory.
type Store struct {
	mu          sync.RWMutex
	databases   map[string]*Database
	tables      map[string]map[string]*Table // dbName -> tableName -> table
	crawlers    map[string]*Crawler
	jobs        map[string]*Job
	jobRuns     map[string][]*JobRun // jobName -> runs
	connections map[string]*Connection
	tags        map[string]map[string]string // arn -> tags
	accountID   string
	region      string
	lcConfig    *lifecycle.Config
	locator     ServiceLocator
}

// NewStore creates a new Glue store.
func NewStore(accountID, region string) *Store {
	return &Store{
		databases:   make(map[string]*Database),
		tables:      make(map[string]map[string]*Table),
		crawlers:    make(map[string]*Crawler),
		jobs:        make(map[string]*Job),
		jobRuns:     make(map[string][]*JobRun),
		connections: make(map[string]*Connection),
		tags:        make(map[string]map[string]string),
		accountID:   accountID,
		region:      region,
		lcConfig:    lifecycle.DefaultConfig(),
	}
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) crawlerARN(name string) string {
	return fmt.Sprintf("arn:aws:glue:%s:%s:crawler/%s", s.region, s.accountID, name)
}

func (s *Store) jobARN(name string) string {
	return fmt.Sprintf("arn:aws:glue:%s:%s:job/%s", s.region, s.accountID, name)
}

func (s *Store) databaseARN(name string) string {
	return fmt.Sprintf("arn:aws:glue:%s:%s:database/%s", s.region, s.accountID, name)
}

func (s *Store) tableARN(db, table string) string {
	return fmt.Sprintf("arn:aws:glue:%s:%s:table/%s/%s", s.region, s.accountID, db, table)
}

func (s *Store) connectionARN(name string) string {
	return fmt.Sprintf("arn:aws:glue:%s:%s:connection/%s", s.region, s.accountID, name)
}

// ---- Database operations ----

func (s *Store) CreateDatabase(name, description, locationURI string) (*Database, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[name]; ok {
		return nil, false
	}
	db := &Database{
		Name:        name,
		Description: description,
		LocationURI: locationURI,
		CreateTime:  time.Now().UTC(),
		Parameters:  make(map[string]string),
	}
	s.databases[name] = db
	s.tables[name] = make(map[string]*Table)
	return db, true
}

func (s *Store) GetDatabase(name string) (*Database, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	db, ok := s.databases[name]
	return db, ok
}

func (s *Store) ListDatabases() []*Database {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Database, 0, len(s.databases))
	for _, db := range s.databases {
		result = append(result, db)
	}
	return result
}

func (s *Store) DeleteDatabase(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[name]; !ok {
		return false
	}
	delete(s.databases, name)
	delete(s.tables, name)
	return true
}

// ---- Table operations ----

func (s *Store) CreateTable(dbName string, table *Table) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tbls, ok := s.tables[dbName]
	if !ok {
		return false
	}
	if _, exists := tbls[table.Name]; exists {
		return false
	}
	table.DatabaseName = dbName
	table.CreateTime = time.Now().UTC()
	table.UpdateTime = table.CreateTime
	tbls[table.Name] = table
	return true
}

func (s *Store) GetTable(dbName, tableName string) (*Table, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tbls, ok := s.tables[dbName]
	if !ok {
		return nil, false
	}
	t, ok := tbls[tableName]
	return t, ok
}

func (s *Store) ListTables(dbName string) []*Table {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tbls, ok := s.tables[dbName]
	if !ok {
		return nil
	}
	result := make([]*Table, 0, len(tbls))
	for _, t := range tbls {
		result = append(result, t)
	}
	return result
}

func (s *Store) UpdateTable(dbName string, table *Table) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tbls, ok := s.tables[dbName]
	if !ok {
		return false
	}
	existing, ok := tbls[table.Name]
	if !ok {
		return false
	}
	table.CreateTime = existing.CreateTime
	table.DatabaseName = dbName
	table.UpdateTime = time.Now().UTC()
	tbls[table.Name] = table
	return true
}

func (s *Store) DeleteTable(dbName, tableName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	tbls, ok := s.tables[dbName]
	if !ok {
		return false
	}
	if _, ok := tbls[tableName]; !ok {
		return false
	}
	delete(tbls, tableName)
	return true
}

// ---- Crawler operations ----

func crawlerTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "RUNNING", To: "READY", Delay: 5 * time.Second},
		{From: "STOPPING", To: "READY", Delay: 2 * time.Second},
	}
}

func (s *Store) CreateCrawler(name, role, dbName, description, schedule string, targets CrawlerTargets, tags map[string]string) (*Crawler, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.crawlers[name]; ok {
		return nil, false
	}
	now := time.Now().UTC()
	c := &Crawler{
		Name:         name,
		Role:         role,
		DatabaseName: dbName,
		Description:  description,
		Targets:      targets,
		Schedule:     schedule,
		State:        "READY",
		Lifecycle:    lifecycle.NewMachine("READY", crawlerTransitions(), s.lcConfig),
		CreateTime:   now,
		LastUpdated:  now,
		Tags:         tags,
	}
	s.crawlers[name] = c
	return c, true
}

func (s *Store) GetCrawler(name string) (*Crawler, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.crawlers[name]
	if ok {
		c.State = string(c.Lifecycle.State())
	}
	return c, ok
}

func (s *Store) ListCrawlers() []*Crawler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Crawler, 0, len(s.crawlers))
	for _, c := range s.crawlers {
		c.State = string(c.Lifecycle.State())
		result = append(result, c)
	}
	return result
}

func (s *Store) DeleteCrawler(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.crawlers[name]
	if !ok {
		return false
	}
	c.Lifecycle.Stop()
	delete(s.crawlers, name)
	return true
}

// SetLocator sets the service locator for cross-service lookups.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
}

func (s *Store) StartCrawler(name string) bool {
	s.mu.Lock()
	c, ok := s.crawlers[name]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if string(c.Lifecycle.State()) != "READY" {
		s.mu.Unlock()
		return false
	}
	crawler := *c // copy for use outside lock
	lc := c.Lifecycle
	locator := s.locator
	s.mu.Unlock()

	if lc != nil {
		lc.ForceState("RUNNING")
	}

	// Behavioral: discover S3 objects and generate mock table schemas
	s.crawlerDiscoverAndCreateTables(&crawler, locator)

	return true
}

// crawlerDiscoverAndCreateTables looks up S3 via locator, discovers objects,
// and creates/updates tables in the Glue database.
func (s *Store) crawlerDiscoverAndCreateTables(crawler *Crawler, locator ServiceLocator) {
	tablesCreated := 0
	tablesUpdated := 0

	for _, target := range crawler.Targets.S3Targets {
		if target.Path == "" {
			continue
		}

		// Try to discover S3 objects via locator
		objectKeys := s.listS3Objects(target.Path, locator)

		// Generate mock table schema from discovered objects
		tableName, columns := s.inferSchemaFromObjects(target.Path, objectKeys)
		if tableName == "" {
			continue
		}

		// Ensure the database exists
		s.mu.Lock()
		if _, ok := s.databases[crawler.DatabaseName]; !ok {
			// Auto-create the database if it doesn't exist
			s.databases[crawler.DatabaseName] = &Database{
				Name:       crawler.DatabaseName,
				CreateTime: time.Now().UTC(),
				Parameters: make(map[string]string),
			}
			s.tables[crawler.DatabaseName] = make(map[string]*Table)
		}

		tbls := s.tables[crawler.DatabaseName]
		now := time.Now().UTC()
		if _, exists := tbls[tableName]; exists {
			// Update existing table
			tbls[tableName].StorageDesc.Columns = columns
			tbls[tableName].StorageDesc.Location = target.Path
			tbls[tableName].UpdateTime = now
			tablesUpdated++
		} else {
			// Create new table
			tbls[tableName] = &Table{
				Name:         tableName,
				DatabaseName: crawler.DatabaseName,
				StorageDesc: StorageDescriptor{
					Location:     target.Path,
					InputFormat:  "org.apache.hadoop.mapred.TextInputFormat",
					OutputFormat: "org.apache.hadoop.hive.ql.io.HiveIgnoreKeyTextOutputFormat",
					Columns:      columns,
				},
				Parameters: map[string]string{"classification": "csv", "crawlerSchemaDeserializerVersion": "1.0"},
				CreateTime: now,
				UpdateTime: now,
			}
			tablesCreated++
		}

		// Update crawler's LastCrawl metadata
		c := s.crawlers[crawler.Name]
		if c != nil {
			c.LastCrawl = &LastCrawlInfo{
				Status:        "SUCCEEDED",
				LogGroup:      "/aws-glue/crawlers",
				LogStream:     crawler.Name,
				MessagePrefix: crawler.Name,
				StartTime:     now,
				TablesCreated: tablesCreated,
				TablesUpdated: tablesUpdated,
			}
		}
		s.mu.Unlock()
	}
}

// listS3Objects queries the S3 service for objects at the given path.
func (s *Store) listS3Objects(path string, locator ServiceLocator) []string {
	if locator == nil {
		return nil
	}
	s3Svc, err := locator.Lookup("s3")
	if err != nil || s3Svc == nil {
		return nil
	}

	// Extract bucket and prefix from s3://bucket/prefix/ path
	trimmed := strings.TrimPrefix(path, "s3://")
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) == 0 {
		return nil
	}
	bucket := parts[0]
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[1]
	}

	body, _ := json.Marshal(map[string]string{
		"Bucket": bucket,
		"Prefix": prefix,
	})
	ctx := &service.RequestContext{
		Action:     "ListObjectsV2",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	resp, err := s3Svc.HandleRequest(ctx)
	if err != nil || resp == nil || resp.Body == nil {
		return nil
	}

	// Parse response
	data, err := json.Marshal(resp.Body)
	if err != nil {
		return nil
	}
	var result struct {
		Contents []struct {
			Key string `json:"Key"`
		} `json:"Contents"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil
	}
	keys := make([]string, len(result.Contents))
	for i, c := range result.Contents {
		keys[i] = c.Key
	}
	return keys
}

// inferSchemaFromObjects generates a table name and columns from S3 object keys.
func (s *Store) inferSchemaFromObjects(path string, objectKeys []string) (string, []Column) {
	// Derive table name from the path
	trimmed := strings.TrimPrefix(path, "s3://")
	trimmed = strings.TrimSuffix(trimmed, "/")
	parts := strings.Split(trimmed, "/")

	tableName := "default_table"
	if len(parts) > 1 {
		tableName = strings.ReplaceAll(parts[len(parts)-1], "-", "_")
	} else if len(parts) == 1 {
		tableName = strings.ReplaceAll(parts[0], "-", "_")
	}

	// Generate mock columns based on object keys or default
	if len(objectKeys) > 0 {
		// Infer columns from the first object key's prefix structure
		columns := []Column{
			{Name: "id", Type: "bigint"},
			{Name: "timestamp", Type: "string"},
			{Name: "data", Type: "string"},
		}
		// Add partition-like columns from prefixes
		for _, key := range objectKeys {
			keyParts := strings.Split(key, "/")
			for _, part := range keyParts {
				if strings.Contains(part, "=") {
					kv := strings.SplitN(part, "=", 2)
					colName := strings.ReplaceAll(kv[0], "-", "_")
					found := false
					for _, c := range columns {
						if c.Name == colName {
							found = true
							break
						}
					}
					if !found {
						columns = append(columns, Column{Name: colName, Type: "string"})
					}
				}
			}
		}
		return tableName, columns
	}

	// Default columns when no S3 objects are available
	return tableName, []Column{
		{Name: "id", Type: "bigint", Comment: "Auto-generated by crawler"},
		{Name: "name", Type: "string", Comment: "Auto-generated by crawler"},
		{Name: "value", Type: "string", Comment: "Auto-generated by crawler"},
		{Name: "created_at", Type: "timestamp", Comment: "Auto-generated by crawler"},
	}
}

func (s *Store) StopCrawler(name string) bool {
	s.mu.Lock()
	c, ok := s.crawlers[name]
	if !ok {
		s.mu.Unlock()
		return false
	}
	if string(c.Lifecycle.State()) != "RUNNING" {
		s.mu.Unlock()
		return false
	}
	lc := c.Lifecycle
	s.mu.Unlock()
	if lc != nil {
		lc.ForceState("STOPPING")
	}
	return true
}

// ---- Job operations ----

func (s *Store) CreateJob(job *Job) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[job.Name]; ok {
		return false
	}
	job.CreateTime = time.Now().UTC()
	s.jobs[job.Name] = job
	s.jobRuns[job.Name] = make([]*JobRun, 0)
	return true
}

func (s *Store) GetJob(name string) (*Job, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	j, ok := s.jobs[name]
	return j, ok
}

func (s *Store) ListJobs() []*Job {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Job, 0, len(s.jobs))
	for _, j := range s.jobs {
		result = append(result, j)
	}
	return result
}

func (s *Store) DeleteJob(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.jobs[name]; !ok {
		return false
	}
	delete(s.jobs, name)
	delete(s.jobRuns, name)
	return true
}

func jobRunTransitions() []lifecycle.Transition {
	return []lifecycle.Transition{
		{From: "STARTING", To: "RUNNING", Delay: 1 * time.Second},
		{From: "RUNNING", To: "SUCCEEDED", Delay: 5 * time.Second},
	}
}

func (s *Store) StartJobRun(jobName string) (*JobRun, bool) {
	s.mu.Lock()
	if _, ok := s.jobs[jobName]; !ok {
		s.mu.Unlock()
		return nil, false
	}
	now := time.Now().UTC()
	run := &JobRun{
		ID:        "jr_" + newUUID(),
		JobName:   jobName,
		State:     "STARTING",
		StartedOn: now,
		Attempt:   len(s.jobRuns[jobName]) + 1,
		Lifecycle: lifecycle.NewMachine("STARTING", jobRunTransitions(), s.lcConfig),
	}
	s.jobRuns[jobName] = append(s.jobRuns[jobName], run)
	locator := s.locator
	s.mu.Unlock()

	// Write mock execution logs via mocklog
	if locator != nil {
		writer := mocklog.NewWriter(locator)
		phases := []string{"INSTALL", "COMPILE", "EXECUTE", "CLEANUP"}
		lines := mocklog.GenerateBuildLines("glue", run.ID, phases)
		writer.WriteBuildLog("/aws-glue/jobs/"+jobName, run.ID, lines)
	}

	return run, true
}

func (s *Store) GetJobRun(jobName, runID string) (*JobRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs, ok := s.jobRuns[jobName]
	if !ok {
		return nil, false
	}
	for _, r := range runs {
		if r.ID == runID {
			r.State = string(r.Lifecycle.State())
			return r, true
		}
	}
	return nil, false
}

func (s *Store) ListJobRuns(jobName string) []*JobRun {
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs, ok := s.jobRuns[jobName]
	if !ok {
		return nil
	}
	result := make([]*JobRun, len(runs))
	for i, r := range runs {
		r.State = string(r.Lifecycle.State())
		result[i] = r
	}
	return result
}

func (s *Store) StopJobRun(jobName, runID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	runs, ok := s.jobRuns[jobName]
	if !ok {
		return false
	}
	for _, r := range runs {
		if r.ID == runID {
			state := string(r.Lifecycle.State())
			if state == "STARTING" || state == "RUNNING" {
				now := time.Now().UTC()
				r.Lifecycle.ForceState("STOPPED")
				r.CompletedOn = &now
			}
			return true
		}
	}
	return false
}

// ---- Connection operations ----

func (s *Store) CreateConnection(conn *Connection) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connections[conn.Name]; ok {
		return false
	}
	conn.CreateTime = time.Now().UTC()
	s.connections[conn.Name] = conn
	return true
}

func (s *Store) GetConnection(name string) (*Connection, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.connections[name]
	return c, ok
}

func (s *Store) ListConnections() []*Connection {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*Connection, 0, len(s.connections))
	for _, c := range s.connections {
		result = append(result, c)
	}
	return result
}

func (s *Store) DeleteConnection(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.connections[name]; !ok {
		return false
	}
	delete(s.connections, name)
	return true
}

// ---- Tag operations ----

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
	if s.tags[arn] == nil {
		return
	}
	for _, k := range keys {
		delete(s.tags[arn], k)
	}
}

func (s *Store) GetTags(arn string) map[string]string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tags := s.tags[arn]
	if tags == nil {
		return make(map[string]string)
	}
	result := make(map[string]string, len(tags))
	for k, v := range tags {
		result[k] = v
	}
	return result
}
