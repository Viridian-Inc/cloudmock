package athena

import (
	crand "crypto/rand"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	"github.com/neureaux/cloudmock/pkg/service"
	"github.com/neureaux/cloudmock/pkg/sqlparse"
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

// QueryResultSet holds mock query results for a completed execution.
type QueryResultSet struct {
	ColumnInfo []ColumnInfo
	Rows       [][]string
}

// ColumnInfo describes a result column.
type ColumnInfo struct {
	Name string
	Type string
}

// DataCatalog represents an Athena data catalog.
type DataCatalog struct {
	Name        string
	Type        string // LAMBDA, GLUE, HIVE
	Description string
	Parameters  map[string]string
	Tags        map[string]string
}

// Store manages all Athena resources in memory.
type Store struct {
	mu              sync.RWMutex
	workGroups      map[string]*WorkGroup
	namedQueries    map[string]*NamedQuery
	queryExecutions map[string]*QueryExecution
	queryResults    map[string]*QueryResultSet // executionID -> results
	dataCatalogs    map[string]*DataCatalog
	schemaRegistry  *sqlparse.SchemaRegistry
	accountID       string
	region          string
	locator         ServiceLocator
}

// ServiceLocator resolves other services by name.
type ServiceLocator interface {
	Lookup(name string) (service.Service, error)
}

// NewStore creates a new Athena Store with the default "primary" workgroup.
func NewStore(accountID, region string) *Store {
	s := &Store{
		workGroups:      make(map[string]*WorkGroup),
		namedQueries:    make(map[string]*NamedQuery),
		queryExecutions: make(map[string]*QueryExecution),
		queryResults:    make(map[string]*QueryResultSet),
		dataCatalogs:    make(map[string]*DataCatalog),
		schemaRegistry:  sqlparse.NewSchemaRegistry(),
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

// SetLocator sets the service locator for cross-service lookups.
func (s *Store) SetLocator(locator ServiceLocator) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.locator = locator
}

// populateSchemaFromGlue attempts to populate the schema registry from the Glue catalog.
func (s *Store) populateSchemaFromGlue(database string) {
	if s.locator == nil {
		return
	}
	glueSvc, err := s.locator.Lookup("glue")
	if err != nil || glueSvc == nil {
		return
	}

	// List tables in the database from Glue
	body, _ := json.Marshal(map[string]string{"DatabaseName": database})
	ctx := &service.RequestContext{
		Action:     "GetTables",
		Body:       body,
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
	}
	resp, err := glueSvc.HandleRequest(ctx)
	if err != nil || resp == nil || resp.Body == nil {
		return
	}

	// Parse response to extract table/column info
	data, err := json.Marshal(resp.Body)
	if err != nil {
		return
	}
	var result struct {
		TableList []struct {
			Name              string `json:"Name"`
			StorageDescriptor struct {
				Columns []struct {
					Name string `json:"Name"`
				} `json:"Columns"`
			} `json:"StorageDescriptor"`
		} `json:"TableList"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return
	}
	for _, tbl := range result.TableList {
		cols := make([]string, len(tbl.StorageDescriptor.Columns))
		for i, c := range tbl.StorageDescriptor.Columns {
			cols[i] = c.Name
		}
		s.schemaRegistry.Register(database, tbl.Name, cols)
	}
}

// GetQueryResultSet returns mock results for a completed query execution.
func (s *Store) GetQueryResultSet(executionID string) (*QueryResultSet, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rs, ok := s.queryResults[executionID]
	return rs, ok
}

func newUUID() string {
	b := make([]byte, 16)
	_, _ = crand.Read(b)
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

// StartQueryExecution creates a new query execution, parses and validates the SQL,
// and generates mock results for SELECT queries.
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

	// Parse and validate SQL
	parsed := sqlparse.Parse(query)
	if !parsed.IsValid {
		completionTime := now.Add(50 * time.Millisecond)
		qe.Status.State = "FAILED"
		qe.Status.CompletionTime = &completionTime
		qe.Status.StateChangeReason = "SYNTAX_ERROR: " + strings.Join(parsed.Errors, "; ")
		qe.Statistics = QueryExecutionStatistics{
			EngineExecutionTimeInMillis: 10,
			TotalExecutionTimeInMillis:  20,
		}
		return qe
	}

	// Populate schema from Glue if database is specified
	if database != "" {
		s.populateSchemaFromGlue(database)
	}

	// Validate against schema registry only if it has schemas registered
	validationErrors := sqlparse.Validate(parsed, s.schemaRegistry)
	if len(validationErrors) > 0 && s.schemaRegistry.Len() > 0 {
		completionTime := now.Add(50 * time.Millisecond)
		qe.Status.State = "FAILED"
		qe.Status.CompletionTime = &completionTime
		qe.Status.StateChangeReason = "SEMANTIC_ERROR: " + strings.Join(validationErrors, "; ")
		qe.Statistics = QueryExecutionStatistics{
			EngineExecutionTimeInMillis: 20,
			TotalExecutionTimeInMillis:  30,
		}
		return qe
	}

	// Compute mock data scanned based on query complexity
	dataScanned := int64(1024) // base
	dataScanned += int64(len(parsed.Tables)) * 4096
	dataScanned += int64(len(parsed.Columns)) * 512

	completionTime := now.Add(100 * time.Millisecond)
	qe.Status.State = "SUCCEEDED"
	qe.Status.CompletionTime = &completionTime
	qe.Statistics = QueryExecutionStatistics{
		EngineExecutionTimeInMillis: 50,
		DataScannedInBytes:          dataScanned,
		TotalExecutionTimeInMillis:  100,
	}

	// Generate mock result set for SELECT queries
	if parsed.StatementType == "SELECT" {
		s.queryResults[qe.ID] = s.generateMockResults(parsed)
	}

	return qe
}

// generateMockResults creates a mock result set based on parsed SQL.
func (s *Store) generateMockResults(parsed *sqlparse.ParseResult) *QueryResultSet {
	columns := parsed.Columns
	if len(columns) == 0 {
		// SELECT * — try to get columns from schema
		for _, tbl := range parsed.Tables {
			if schema, ok := s.schemaRegistry.Lookup(tbl); ok {
				columns = schema.Columns
				break
			}
		}
	}
	if len(columns) == 0 {
		columns = []string{"col1", "col2", "col3"}
	}

	colInfo := make([]ColumnInfo, len(columns))
	for i, c := range columns {
		colInfo[i] = ColumnInfo{Name: c, Type: "varchar"}
	}

	// Generate 5-10 mock rows
	numRows := 5 + rand.IntN(6)
	rows := make([][]string, numRows)
	for i := range rows {
		row := make([]string, len(columns))
		for j, col := range columns {
			row[j] = fmt.Sprintf("%s_%d", col, i+1)
		}
		rows[i] = row
	}

	return &QueryResultSet{
		ColumnInfo: colInfo,
		Rows:       rows,
	}
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

// ListTagsForResource returns tags for the given resource ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, wg := range s.workGroups {
		if s.workGroupARN(wg.Name) == arn {
			result := make(map[string]string, len(wg.Tags))
			for k, v := range wg.Tags {
				result[k] = v
			}
			return result, true
		}
	}
	// Check data catalogs
	for _, dc := range s.dataCatalogs {
		if s.dataCatalogARN(dc.Name) == arn {
			result := make(map[string]string, len(dc.Tags))
			for k, v := range dc.Tags {
				result[k] = v
			}
			return result, true
		}
	}
	return nil, false
}

// BatchGetNamedQuery returns multiple named queries by ID.
func (s *Store) BatchGetNamedQuery(ids []string) ([]*NamedQuery, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	found := make([]*NamedQuery, 0)
	notFound := make([]string, 0)
	for _, id := range ids {
		nq, ok := s.namedQueries[id]
		if ok {
			found = append(found, nq)
		} else {
			notFound = append(notFound, id)
		}
	}
	return found, notFound
}

// BatchGetQueryExecution returns multiple query executions by ID.
func (s *Store) BatchGetQueryExecution(ids []string) ([]*QueryExecution, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	found := make([]*QueryExecution, 0)
	notFound := make([]string, 0)
	for _, id := range ids {
		qe, ok := s.queryExecutions[id]
		if ok {
			found = append(found, qe)
		} else {
			notFound = append(notFound, id)
		}
	}
	return found, notFound
}

// ---- DataCatalog operations ----

func (s *Store) dataCatalogARN(name string) string {
	return fmt.Sprintf("arn:aws:athena:%s:%s:datacatalog/%s", s.region, s.accountID, name)
}

// CreateDataCatalog creates a new data catalog. Returns false if already exists.
func (s *Store) CreateDataCatalog(name, catalogType, description string, params, tags map[string]string) (*DataCatalog, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataCatalogs[name]; ok {
		return nil, false
	}
	dc := &DataCatalog{
		Name:        name,
		Type:        catalogType,
		Description: description,
		Parameters:  params,
		Tags:        tags,
	}
	s.dataCatalogs[name] = dc
	return dc, true
}

// GetDataCatalog returns a data catalog by name.
func (s *Store) GetDataCatalog(name string) (*DataCatalog, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dc, ok := s.dataCatalogs[name]
	return dc, ok
}

// ListDataCatalogs returns all data catalogs.
func (s *Store) ListDataCatalogs() []*DataCatalog {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*DataCatalog, 0, len(s.dataCatalogs))
	for _, dc := range s.dataCatalogs {
		result = append(result, dc)
	}
	return result
}

// UpdateDataCatalog updates a data catalog's description and parameters.
func (s *Store) UpdateDataCatalog(name, description string, params map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	dc, ok := s.dataCatalogs[name]
	if !ok {
		return false
	}
	if description != "" {
		dc.Description = description
	}
	if params != nil {
		for k, v := range params {
			dc.Parameters[k] = v
		}
	}
	return true
}

// DeleteDataCatalog deletes a data catalog. Returns false if not found.
func (s *Store) DeleteDataCatalog(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.dataCatalogs[name]; !ok {
		return false
	}
	delete(s.dataCatalogs, name)
	return true
}
