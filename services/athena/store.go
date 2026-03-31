package athena

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// WorkGroup represents an Athena workgroup.
type WorkGroup struct {
	Name          string
	Description   string
	State         string // ENABLED or DISABLED
	CreationTime  time.Time
	Tags          map[string]string
	Configuration WorkGroupConfiguration
}

// WorkGroupConfiguration holds workgroup settings.
type WorkGroupConfiguration struct {
	ResultOutputLocation       string
	EnforceWorkGroupConfiguration bool
	BytesScannedCutoffPerQuery int64
}

// NamedQuery represents an Athena named query.
type NamedQuery struct {
	ID          string
	Name        string
	Description string
	Database    string
	QueryString string
	WorkGroup   string
}

// QueryExecution represents a running or completed query.
type QueryExecution struct {
	ID              string
	Query           string
	Database        string
	WorkGroup       string
	Status          QueryExecutionStatus
	ResultConfig    ResultConfiguration
	Statistics      QueryExecutionStatistics
	SubmissionTime  time.Time
}

// QueryExecutionStatus holds query state.
type QueryExecutionStatus struct {
	State            string // QUEUED, RUNNING, SUCCEEDED, FAILED, CANCELLED
	StateChangeReason string
	SubmissionTime   time.Time
	CompletionTime   *time.Time
}

// ResultConfiguration holds result output settings.
type ResultConfiguration struct {
	OutputLocation string
}

// QueryExecutionStatistics holds execution stats.
type QueryExecutionStatistics struct {
	EngineExecutionTimeInMillis int64
	DataScannedInBytes          int64
	TotalExecutionTimeInMillis  int64
}

// Store manages all Athena resources in memory.
type Store struct {
	mu              sync.RWMutex
	workGroups      map[string]*WorkGroup
	namedQueries    map[string]*NamedQuery
	queryExecutions map[string]*QueryExecution
	accountID       string
	region          string
}

// NewStore creates a new Athena Store with the default "primary" workgroup.
func NewStore(accountID, region string) *Store {
	s := &Store{
		workGroups:      make(map[string]*WorkGroup),
		namedQueries:    make(map[string]*NamedQuery),
		queryExecutions: make(map[string]*QueryExecution),
		accountID:       accountID,
		region:          region,
	}
	s.workGroups["primary"] = &WorkGroup{
		Name:         "primary",
		State:        "ENABLED",
		CreationTime: time.Now().UTC(),
		Tags:         make(map[string]string),
	}
	return s
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (s *Store) workGroupARN(name string) string {
	return fmt.Sprintf("arn:aws:athena:%s:%s:workgroup/%s", s.region, s.accountID, name)
}

// CreateWorkGroup creates a new workgroup. Returns false if it already exists.
func (s *Store) CreateWorkGroup(name, description string, tags map[string]string, config WorkGroupConfiguration) (*WorkGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workGroups[name]; ok {
		return nil, false
	}
	wg := &WorkGroup{
		Name:          name,
		Description:   description,
		State:         "ENABLED",
		CreationTime:  time.Now().UTC(),
		Tags:          tags,
		Configuration: config,
	}
	s.workGroups[name] = wg
	return wg, true
}

// GetWorkGroup returns a workgroup by name.
func (s *Store) GetWorkGroup(name string) (*WorkGroup, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	wg, ok := s.workGroups[name]
	return wg, ok
}

// ListWorkGroups returns all workgroups.
func (s *Store) ListWorkGroups() []*WorkGroup {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*WorkGroup, 0, len(s.workGroups))
	for _, wg := range s.workGroups {
		result = append(result, wg)
	}
	return result
}

// UpdateWorkGroup updates a workgroup's description and state.
func (s *Store) UpdateWorkGroup(name, description, state string) (*WorkGroup, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	wg, ok := s.workGroups[name]
	if !ok {
		return nil, false
	}
	if description != "" {
		wg.Description = description
	}
	if state != "" {
		wg.State = state
	}
	return wg, true
}

// DeleteWorkGroup deletes a workgroup. Returns false if not found.
func (s *Store) DeleteWorkGroup(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.workGroups[name]; !ok {
		return false
	}
	delete(s.workGroups, name)
	return true
}

// CreateNamedQuery creates a named query. Returns the query ID.
func (s *Store) CreateNamedQuery(name, description, database, queryString, workGroup string) *NamedQuery {
	s.mu.Lock()
	defer s.mu.Unlock()
	if workGroup == "" {
		workGroup = "primary"
	}
	nq := &NamedQuery{
		ID:          newUUID(),
		Name:        name,
		Description: description,
		Database:    database,
		QueryString: queryString,
		WorkGroup:   workGroup,
	}
	s.namedQueries[nq.ID] = nq
	return nq
}

// GetNamedQuery returns a named query by ID.
func (s *Store) GetNamedQuery(id string) (*NamedQuery, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	nq, ok := s.namedQueries[id]
	return nq, ok
}

// ListNamedQueries returns all named query IDs, optionally filtered by workgroup.
func (s *Store) ListNamedQueries(workGroup string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0)
	for _, nq := range s.namedQueries {
		if workGroup == "" || nq.WorkGroup == workGroup {
			ids = append(ids, nq.ID)
		}
	}
	return ids
}

// DeleteNamedQuery deletes a named query. Returns false if not found.
func (s *Store) DeleteNamedQuery(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.namedQueries[id]; !ok {
		return false
	}
	delete(s.namedQueries, id)
	return true
}

// StartQueryExecution creates a new query execution in QUEUED state.
func (s *Store) StartQueryExecution(query, database, workGroup, outputLocation string) *QueryExecution {
	s.mu.Lock()
	defer s.mu.Unlock()
	if workGroup == "" {
		workGroup = "primary"
	}
	now := time.Now().UTC()
	qe := &QueryExecution{
		ID:        newUUID(),
		Query:     query,
		Database:  database,
		WorkGroup: workGroup,
		Status: QueryExecutionStatus{
			State:          "QUEUED",
			SubmissionTime: now,
		},
		ResultConfig: ResultConfiguration{
			OutputLocation: outputLocation,
		},
		SubmissionTime: now,
	}
	s.queryExecutions[qe.ID] = qe
	// Immediately transition to SUCCEEDED for mock purposes.
	completionTime := now.Add(100 * time.Millisecond)
	qe.Status.State = "SUCCEEDED"
	qe.Status.CompletionTime = &completionTime
	qe.Statistics = QueryExecutionStatistics{
		EngineExecutionTimeInMillis: 50,
		DataScannedInBytes:          1024,
		TotalExecutionTimeInMillis:  100,
	}
	return qe
}

// GetQueryExecution returns a query execution by ID.
func (s *Store) GetQueryExecution(id string) (*QueryExecution, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	qe, ok := s.queryExecutions[id]
	return qe, ok
}

// ListQueryExecutions returns all query execution IDs, optionally filtered by workgroup.
func (s *Store) ListQueryExecutions(workGroup string) []string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ids := make([]string, 0)
	for _, qe := range s.queryExecutions {
		if workGroup == "" || qe.WorkGroup == workGroup {
			ids = append(ids, qe.ID)
		}
	}
	return ids
}

// StopQueryExecution cancels a query execution.
func (s *Store) StopQueryExecution(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	qe, ok := s.queryExecutions[id]
	if !ok {
		return false
	}
	if qe.Status.State == "QUEUED" || qe.Status.State == "RUNNING" {
		now := time.Now().UTC()
		qe.Status.State = "CANCELLED"
		qe.Status.CompletionTime = &now
		qe.Status.StateChangeReason = "Query was cancelled"
	}
	return true
}

// TagResource adds tags to a workgroup ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wg := range s.workGroups {
		if s.workGroupARN(wg.Name) == arn {
			for k, v := range tags {
				wg.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tag keys from a workgroup ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, wg := range s.workGroups {
		if s.workGroupARN(wg.Name) == arn {
			for _, k := range keys {
				delete(wg.Tags, k)
			}
			return true
		}
	}
	return false
}
