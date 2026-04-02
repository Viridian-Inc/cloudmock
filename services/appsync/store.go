package appsync

import (
	"crypto/rand"
	"fmt"
	"sync"
	"time"
)

// GraphqlApi represents an AppSync GraphQL API.
type GraphqlApi struct {
	ApiId               string
	Name                string
	ARN                 string
	AuthenticationType  string
	LogConfig           map[string]any
	UserPoolConfig      map[string]any
	OpenIDConnectConfig map[string]any
	AdditionalAuth      []map[string]any
	XrayEnabled         bool
	Uris                map[string]string
	Tags                map[string]string
	CreatedAt           time.Time
}

// DataSource represents an AppSync data source.
type DataSource struct {
	DataSourceArn string
	ApiId         string
	Name          string
	Type          string
	Description   string
	ServiceRoleArn string
	DynamodbConfig  map[string]any
	LambdaConfig    map[string]any
	HttpConfig      map[string]any
	ElasticsearchConfig map[string]any
}

// Resolver represents an AppSync resolver.
type Resolver struct {
	ApiId                  string
	TypeName               string
	FieldName              string
	DataSourceName         string
	RequestMappingTemplate string
	ResponseMappingTemplate string
	Kind                   string
	PipelineConfig         map[string]any
	CachingConfig          map[string]any
	Runtime                map[string]any
	Code                   string
}

// Function represents an AppSync function.
type Function struct {
	FunctionId              string
	ApiId                   string
	Name                    string
	Description             string
	DataSourceName          string
	RequestMappingTemplate  string
	ResponseMappingTemplate string
	FunctionVersion         string
	Runtime                 map[string]any
	Code                    string
}

// ApiKey represents an AppSync API key.
type ApiKey struct {
	Id          string
	ApiId       string
	Description string
	Expires     int64
	Deletes     int64
}

// TypeDef represents an AppSync type definition.
type TypeDef struct {
	ApiId       string
	Name        string
	Format      string
	Definition  string
	Description string
	ARN         string
}

// SchemaCreation tracks schema creation state.
type SchemaCreation struct {
	Status  string // PROCESSING | ACTIVE | FAILED | DELETING
	Details string
}

// Store manages all AppSync state in memory.
type Store struct {
	mu          sync.RWMutex
	apis        map[string]*GraphqlApi
	dataSources map[string]map[string]*DataSource  // apiID -> name -> ds
	resolvers   map[string]map[string]*Resolver    // apiID -> typeName.fieldName -> resolver
	functions   map[string]map[string]*Function    // apiID -> funcID -> function
	apiKeys     map[string]map[string]*ApiKey      // apiID -> keyID -> key
	types       map[string]map[string]*TypeDef     // apiID -> name -> type
	schemas     map[string]*SchemaCreation         // apiID -> schema creation status
	accountID   string
	region      string
	nextFuncNum int
}

// NewStore returns a new Store for the given account and region.
func NewStore(accountID, region string) *Store {
	return &Store{
		apis:        make(map[string]*GraphqlApi),
		dataSources: make(map[string]map[string]*DataSource),
		resolvers:   make(map[string]map[string]*Resolver),
		functions:   make(map[string]map[string]*Function),
		apiKeys:     make(map[string]map[string]*ApiKey),
		types:       make(map[string]map[string]*TypeDef),
		schemas:     make(map[string]*SchemaCreation),
		accountID:   accountID,
		region:      region,
		nextFuncNum: 1,
	}
}

func newID() string {
	b := make([]byte, 13)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)[:26]
}

func newAPIKeyID() string {
	b := make([]byte, 20)
	_, _ = rand.Read(b)
	return fmt.Sprintf("da2-%x", b)[:38]
}

func (s *Store) apiARN(apiID string) string {
	return fmt.Sprintf("arn:aws:appsync:%s:%s:apis/%s", s.region, s.accountID, apiID)
}

func (s *Store) dsARN(apiID, name string) string {
	return fmt.Sprintf("arn:aws:appsync:%s:%s:apis/%s/datasources/%s", s.region, s.accountID, apiID, name)
}

func resolverKey(typeName, fieldName string) string {
	return typeName + "." + fieldName
}

// CreateGraphqlApi creates a new GraphQL API.
func (s *Store) CreateGraphqlApi(name, authType string, tags map[string]string, xrayEnabled bool) *GraphqlApi {
	s.mu.Lock()
	defer s.mu.Unlock()
	if tags == nil {
		tags = make(map[string]string)
	}
	if authType == "" {
		authType = "API_KEY"
	}
	id := newID()
	api := &GraphqlApi{
		ApiId: id, Name: name, ARN: s.apiARN(id),
		AuthenticationType: authType, XrayEnabled: xrayEnabled,
		Uris: map[string]string{
			"GRAPHQL": fmt.Sprintf("https://%s.appsync-api.%s.amazonaws.com/graphql", id, s.region),
			"REALTIME": fmt.Sprintf("wss://%s.appsync-realtime-api.%s.amazonaws.com/graphql", id, s.region),
		},
		Tags: tags, CreatedAt: time.Now().UTC(),
	}
	s.apis[id] = api
	s.dataSources[id] = make(map[string]*DataSource)
	s.resolvers[id] = make(map[string]*Resolver)
	s.functions[id] = make(map[string]*Function)
	s.apiKeys[id] = make(map[string]*ApiKey)
	return api
}

// GetGraphqlApi returns a GraphQL API by ID.
func (s *Store) GetGraphqlApi(apiID string) (*GraphqlApi, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	api, ok := s.apis[apiID]
	return api, ok
}

// ListGraphqlApis returns all GraphQL APIs.
func (s *Store) ListGraphqlApis() []*GraphqlApi {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*GraphqlApi, 0, len(s.apis))
	for _, api := range s.apis {
		result = append(result, api)
	}
	return result
}

// UpdateGraphqlApi updates a GraphQL API.
func (s *Store) UpdateGraphqlApi(apiID, name, authType string, xrayEnabled *bool) (*GraphqlApi, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	api, ok := s.apis[apiID]
	if !ok {
		return nil, false
	}
	if name != "" {
		api.Name = name
	}
	if authType != "" {
		api.AuthenticationType = authType
	}
	if xrayEnabled != nil {
		api.XrayEnabled = *xrayEnabled
	}
	return api, true
}

// DeleteGraphqlApi removes a GraphQL API.
func (s *Store) DeleteGraphqlApi(apiID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apis[apiID]; !ok {
		return false
	}
	delete(s.apis, apiID)
	delete(s.dataSources, apiID)
	delete(s.resolvers, apiID)
	delete(s.functions, apiID)
	delete(s.apiKeys, apiID)
	return true
}

// CreateDataSource creates a new data source.
func (s *Store) CreateDataSource(apiID, name, dsType, description, serviceRoleArn string, dynamodb, lambda, httpConf, es map[string]any) (*DataSource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dss, ok := s.dataSources[apiID]
	if !ok {
		return nil, false
	}
	if _, exists := dss[name]; exists {
		return nil, false
	}
	ds := &DataSource{
		DataSourceArn: s.dsARN(apiID, name), ApiId: apiID, Name: name,
		Type: dsType, Description: description, ServiceRoleArn: serviceRoleArn,
		DynamodbConfig: dynamodb, LambdaConfig: lambda,
		HttpConfig: httpConf, ElasticsearchConfig: es,
	}
	dss[name] = ds
	return ds, true
}

// GetDataSource returns a data source.
func (s *Store) GetDataSource(apiID, name string) (*DataSource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dss, ok := s.dataSources[apiID]
	if !ok {
		return nil, false
	}
	ds, ok := dss[name]
	return ds, ok
}

// ListDataSources returns all data sources for an API.
func (s *Store) ListDataSources(apiID string) ([]*DataSource, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	dss, ok := s.dataSources[apiID]
	if !ok {
		return nil, false
	}
	result := make([]*DataSource, 0, len(dss))
	for _, ds := range dss {
		result = append(result, ds)
	}
	return result, true
}

// UpdateDataSource updates a data source.
func (s *Store) UpdateDataSource(apiID, name, dsType, description, serviceRoleArn string) (*DataSource, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	dss, ok := s.dataSources[apiID]
	if !ok {
		return nil, false
	}
	ds, ok := dss[name]
	if !ok {
		return nil, false
	}
	if dsType != "" {
		ds.Type = dsType
	}
	if description != "" {
		ds.Description = description
	}
	if serviceRoleArn != "" {
		ds.ServiceRoleArn = serviceRoleArn
	}
	return ds, true
}

// DeleteDataSource removes a data source.
func (s *Store) DeleteDataSource(apiID, name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	dss, ok := s.dataSources[apiID]
	if !ok {
		return false
	}
	if _, ok := dss[name]; !ok {
		return false
	}
	delete(dss, name)
	return true
}

// CreateResolver creates a new resolver.
func (s *Store) CreateResolver(apiID, typeName, fieldName, dataSourceName, requestTemplate, responseTemplate, kind, code string, pipelineConfig, cachingConfig, runtime map[string]any) (*Resolver, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resolvers, ok := s.resolvers[apiID]
	if !ok {
		return nil, false
	}
	key := resolverKey(typeName, fieldName)
	if _, exists := resolvers[key]; exists {
		return nil, false
	}
	if kind == "" {
		kind = "UNIT"
	}
	r := &Resolver{
		ApiId: apiID, TypeName: typeName, FieldName: fieldName,
		DataSourceName: dataSourceName,
		RequestMappingTemplate: requestTemplate, ResponseMappingTemplate: responseTemplate,
		Kind: kind, PipelineConfig: pipelineConfig, CachingConfig: cachingConfig,
		Runtime: runtime, Code: code,
	}
	resolvers[key] = r
	return r, true
}

// GetResolver returns a resolver.
func (s *Store) GetResolver(apiID, typeName, fieldName string) (*Resolver, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resolvers, ok := s.resolvers[apiID]
	if !ok {
		return nil, false
	}
	r, ok := resolvers[resolverKey(typeName, fieldName)]
	return r, ok
}

// ListResolvers returns all resolvers for a type.
func (s *Store) ListResolvers(apiID, typeName string) ([]*Resolver, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	resolvers, ok := s.resolvers[apiID]
	if !ok {
		return nil, false
	}
	result := make([]*Resolver, 0)
	for _, r := range resolvers {
		if typeName == "" || r.TypeName == typeName {
			result = append(result, r)
		}
	}
	return result, true
}

// UpdateResolver updates a resolver.
func (s *Store) UpdateResolver(apiID, typeName, fieldName, dataSourceName, requestTemplate, responseTemplate, kind, code string) (*Resolver, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resolvers, ok := s.resolvers[apiID]
	if !ok {
		return nil, false
	}
	key := resolverKey(typeName, fieldName)
	r, ok := resolvers[key]
	if !ok {
		return nil, false
	}
	if dataSourceName != "" {
		r.DataSourceName = dataSourceName
	}
	if requestTemplate != "" {
		r.RequestMappingTemplate = requestTemplate
	}
	if responseTemplate != "" {
		r.ResponseMappingTemplate = responseTemplate
	}
	if kind != "" {
		r.Kind = kind
	}
	if code != "" {
		r.Code = code
	}
	return r, true
}

// DeleteResolver removes a resolver.
func (s *Store) DeleteResolver(apiID, typeName, fieldName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	resolvers, ok := s.resolvers[apiID]
	if !ok {
		return false
	}
	key := resolverKey(typeName, fieldName)
	if _, ok := resolvers[key]; !ok {
		return false
	}
	delete(resolvers, key)
	return true
}

// CreateFunction creates a new function.
func (s *Store) CreateFunction(apiID, name, description, dataSourceName, requestTemplate, responseTemplate, functionVersion, code string, runtime map[string]any) (*Function, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	funcs, ok := s.functions[apiID]
	if !ok {
		return nil, false
	}
	if functionVersion == "" {
		functionVersion = "2018-05-29"
	}
	id := fmt.Sprintf("%d", s.nextFuncNum)
	s.nextFuncNum++
	f := &Function{
		FunctionId: id, ApiId: apiID, Name: name, Description: description,
		DataSourceName: dataSourceName,
		RequestMappingTemplate: requestTemplate, ResponseMappingTemplate: responseTemplate,
		FunctionVersion: functionVersion, Runtime: runtime, Code: code,
	}
	funcs[id] = f
	return f, true
}

// GetFunction returns a function by ID.
func (s *Store) GetFunction(apiID, functionID string) (*Function, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	funcs, ok := s.functions[apiID]
	if !ok {
		return nil, false
	}
	f, ok := funcs[functionID]
	return f, ok
}

// ListFunctions returns all functions for an API.
func (s *Store) ListFunctions(apiID string) ([]*Function, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	funcs, ok := s.functions[apiID]
	if !ok {
		return nil, false
	}
	result := make([]*Function, 0, len(funcs))
	for _, f := range funcs {
		result = append(result, f)
	}
	return result, true
}

// UpdateFunction updates a function.
func (s *Store) UpdateFunction(apiID, functionID, name, description, dataSourceName, requestTemplate, responseTemplate, code string) (*Function, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	funcs, ok := s.functions[apiID]
	if !ok {
		return nil, false
	}
	f, ok := funcs[functionID]
	if !ok {
		return nil, false
	}
	if name != "" {
		f.Name = name
	}
	if description != "" {
		f.Description = description
	}
	if dataSourceName != "" {
		f.DataSourceName = dataSourceName
	}
	if requestTemplate != "" {
		f.RequestMappingTemplate = requestTemplate
	}
	if responseTemplate != "" {
		f.ResponseMappingTemplate = responseTemplate
	}
	if code != "" {
		f.Code = code
	}
	return f, true
}

// DeleteFunction removes a function.
func (s *Store) DeleteFunction(apiID, functionID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	funcs, ok := s.functions[apiID]
	if !ok {
		return false
	}
	if _, ok := funcs[functionID]; !ok {
		return false
	}
	delete(funcs, functionID)
	return true
}

// CreateApiKey creates a new API key.
func (s *Store) CreateApiKey(apiID, description string, expires int64) (*ApiKey, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, ok := s.apiKeys[apiID]
	if !ok {
		return nil, false
	}
	if expires == 0 {
		expires = time.Now().Add(7 * 24 * time.Hour).Unix()
	}
	id := newAPIKeyID()
	key := &ApiKey{
		Id: id, ApiId: apiID, Description: description,
		Expires: expires, Deletes: expires + 60*24*60*60,
	}
	keys[id] = key
	return key, true
}

// ListApiKeys returns all API keys for an API.
func (s *Store) ListApiKeys(apiID string) ([]*ApiKey, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	keys, ok := s.apiKeys[apiID]
	if !ok {
		return nil, false
	}
	result := make([]*ApiKey, 0, len(keys))
	for _, k := range keys {
		result = append(result, k)
	}
	return result, true
}

// UpdateApiKey updates an API key.
func (s *Store) UpdateApiKey(apiID, keyID, description string, expires int64) (*ApiKey, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, ok := s.apiKeys[apiID]
	if !ok {
		return nil, false
	}
	key, ok := keys[keyID]
	if !ok {
		return nil, false
	}
	if description != "" {
		key.Description = description
	}
	if expires > 0 {
		key.Expires = expires
	}
	return key, true
}

// DeleteApiKey removes an API key.
func (s *Store) DeleteApiKey(apiID, keyID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	keys, ok := s.apiKeys[apiID]
	if !ok {
		return false
	}
	if _, ok := keys[keyID]; !ok {
		return false
	}
	delete(keys, keyID)
	return true
}

// TagResource applies tags to an API by ARN.
func (s *Store) TagResource(arn string, tags map[string]string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, api := range s.apis {
		if api.ARN == arn {
			for k, v := range tags {
				api.Tags[k] = v
			}
			return true
		}
	}
	return false
}

// UntagResource removes tags from an API by ARN.
func (s *Store) UntagResource(arn string, keys []string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, api := range s.apis {
		if api.ARN == arn {
			for _, k := range keys {
				delete(api.Tags, k)
			}
			return true
		}
	}
	return false
}

// ListTagsForResource returns tags for an API by ARN.
func (s *Store) ListTagsForResource(arn string) (map[string]string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, api := range s.apis {
		if api.ARN == arn {
			cp := make(map[string]string, len(api.Tags))
			for k, v := range api.Tags {
				cp[k] = v
			}
			return cp, true
		}
	}
	return nil, false
}

// ---- Type methods ----

// CreateType creates a new type definition for an API.
func (s *Store) CreateType(apiID, name, format, definition, description string) (*TypeDef, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apis[apiID]; !ok {
		return nil, false
	}
	if _, ok := s.types[apiID]; !ok {
		s.types[apiID] = make(map[string]*TypeDef)
	}
	td := &TypeDef{
		ApiId:       apiID,
		Name:        name,
		Format:      format,
		Definition:  definition,
		Description: description,
		ARN:         fmt.Sprintf("arn:aws:appsync:%s:%s:apis/%s/types/%s", s.region, s.accountID, apiID, name),
	}
	s.types[apiID][name] = td
	return td, true
}

// GetType returns a type definition.
func (s *Store) GetType(apiID, typeName, format string) (*TypeDef, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if ts, ok := s.types[apiID]; ok {
		if td, ok := ts[typeName]; ok {
			return td, true
		}
	}
	return nil, false
}

// ListTypes returns all types for an API.
func (s *Store) ListTypes(apiID, format string) []*TypeDef {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ts, ok := s.types[apiID]
	if !ok {
		return nil
	}
	result := make([]*TypeDef, 0, len(ts))
	for _, td := range ts {
		if format == "" || td.Format == format {
			result = append(result, td)
		}
	}
	return result
}

// UpdateType updates a type definition.
func (s *Store) UpdateType(apiID, typeName, format, definition, description string) (*TypeDef, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ts, ok := s.types[apiID]
	if !ok {
		return nil, false
	}
	td, ok := ts[typeName]
	if !ok {
		return nil, false
	}
	if format != "" {
		td.Format = format
	}
	if definition != "" {
		td.Definition = definition
	}
	if description != "" {
		td.Description = description
	}
	return td, true
}

// DeleteType removes a type definition.
func (s *Store) DeleteType(apiID, typeName string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	ts, ok := s.types[apiID]
	if !ok {
		return false
	}
	if _, ok := ts[typeName]; !ok {
		return false
	}
	delete(ts, typeName)
	return true
}

// ---- Schema methods ----

// StartSchemaCreation begins schema creation for an API.
func (s *Store) StartSchemaCreation(apiID, definition string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apis[apiID]; !ok {
		return false
	}
	// In mock mode, schema is immediately ACTIVE.
	s.schemas[apiID] = &SchemaCreation{Status: "ACTIVE", Details: "Schema created successfully."}
	// Also extract simple type names from SDL if possible (simplified).
	if _, ok := s.types[apiID]; !ok {
		s.types[apiID] = make(map[string]*TypeDef)
	}
	return true
}

// GetSchemaCreationStatus returns the schema creation status for an API.
func (s *Store) GetSchemaCreationStatus(apiID string) (*SchemaCreation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sc, ok := s.schemas[apiID]
	return sc, ok
}
