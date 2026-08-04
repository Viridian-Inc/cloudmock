package cosmosdb

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var cosmosAPIVersions = []string{"2025-05-01", "2024-05-15"}

// CosmosDBService implements first-slice Azure Cosmos DB resource-provider APIs.
type CosmosDBService struct {
	mu              sync.RWMutex
	accounts        map[string]DatabaseAccount
	databases       map[string]SQLDatabase
	containers      map[string]SQLContainer
	dataDatabases   map[string]cosmosDataDatabase
	dataCollections map[string]cosmosDataCollection
	dataDocuments   map[string]cosmosDataDocument
	nextID          uint64
}

func New() *CosmosDBService {
	return &CosmosDBService{
		accounts:        make(map[string]DatabaseAccount),
		databases:       make(map[string]SQLDatabase),
		containers:      make(map[string]SQLContainer),
		dataDatabases:   make(map[string]cosmosDataDatabase),
		dataCollections: make(map[string]cosmosDataCollection),
		dataDocuments:   make(map[string]cosmosDataDocument),
	}
}

func (s *CosmosDBService) Name() string { return "Microsoft.DocumentDB" }

func (s *CosmosDBService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateDatabaseAccount", Method: http.MethodPut, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/write"},
		{Name: "GetDatabaseAccount", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/read"},
		{Name: "ListDatabaseAccounts", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/read"},
		{Name: "DeleteDatabaseAccount", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/delete"},
		{Name: "CreateOrUpdateSQLDatabase", Method: http.MethodPut, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/write"},
		{Name: "GetSQLDatabase", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/read"},
		{Name: "ListSQLDatabases", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/read"},
		{Name: "DeleteSQLDatabase", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/delete"},
		{Name: "CreateOrUpdateSQLContainer", Method: http.MethodPut, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/write"},
		{Name: "GetSQLContainer", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/read"},
		{Name: "ListSQLContainers", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/read"},
		{Name: "DeleteSQLContainer", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers/delete"},
		{Name: "CreateDatabase", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/databases/write"},
		{Name: "ListDatabases", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/databases/read"},
		{Name: "CreateCollection", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/collections/write"},
		{Name: "ListPartitionKeyRanges", Method: http.MethodGet, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/partitionKeyRanges/read"},
		{Name: "CreateDocument", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/documents/write"},
		{Name: "QueryDocuments", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/documents/query"},
		{Name: "GetQueryPlan", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/documents/queryPlan"},
		{Name: "ExecuteTransactionalBatch", Method: http.MethodPost, IAMAction: "azure:Microsoft.DocumentDB/sqlApi/documents/batch"},
	}
}

func (s *CosmosDBService) HealthCheck() error { return nil }

func (s *CosmosDBService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(cosmosAPIVersions))
	for _, apiVersion := range cosmosAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.DocumentDB/databaseAccounts",
			APIVersion: apiVersion,
		})
	}
	keys = append(keys, routing.ServiceKey{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.DocumentDB/sqlApi",
		APIVersion: cosmosSQLDataPlaneAPIVersion,
	})
	return keys
}

func (s *CosmosDBService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts") ||
		strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases") ||
		strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers")
}

func (s *CosmosDBService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Cosmos DB template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Cosmos DB template resource is missing name")
	}
	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
		"options":    resource["options"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	switch {
	case strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts"):
		resp, err = s.createOrUpdateAccount(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases"):
		accountName, databaseName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Cosmos SQL database template resource name must be {account}/{database}")
		}
		resp, err = s.createOrUpdateDatabase(subscriptionID, resourceGroup, accountName, databaseName, data)
	case strings.EqualFold(resourceType, "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers"):
		accountName, databaseName, containerName, ok := splitTripleNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Cosmos SQL container template resource name must be {account}/{database}/{container}")
		}
		resp, err = s.createOrUpdateContainer(subscriptionID, resourceGroup, accountName, databaseName, containerName, data)
	default:
		err = fmt.Errorf("unsupported Cosmos DB template resource type %q", resourceType)
	}
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *CosmosDBService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if cosmosSQLDataPlaneAccount(ctx.RawRequest) != "" {
		return s.handleSQLDataPlane(ctx)
	}
	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "databaseAccounts") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Cosmos DB route is not implemented.")
	}
	if route.DatabaseType != "" {
		if strings.EqualFold(route.DatabaseType, "sqlDatabases") {
			if route.ContainerType != "" {
				if strings.EqualFold(route.ContainerType, "containers") {
					return s.handleContainerRequest(ctx, route)
				}
				return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Cosmos DB route is not implemented.")
			}
			return s.handleDatabaseRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Cosmos DB route is not implemented.")
	}
	return s.handleAccountRequest(ctx, route)
}

func (s *CosmosDBService) handleAccountRequest(ctx *service.RequestContext, route cosmosRoute) (*service.Response, error) {
	if route.AccountName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listAccounts(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateAccount(route.SubscriptionID, route.ResourceGroup, route.AccountName, ctx.Body)
	case http.MethodGet:
		return s.getAccount(route.SubscriptionID, route.ResourceGroup, route.AccountName)
	case http.MethodDelete:
		return s.deleteAccount(route.SubscriptionID, route.ResourceGroup, route.AccountName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CosmosDBService) handleDatabaseRequest(ctx *service.RequestContext, route cosmosRoute) (*service.Response, error) {
	if route.DatabaseName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listDatabases(route.SubscriptionID, route.ResourceGroup, route.AccountName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateDatabase(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName, ctx.Body)
	case http.MethodGet:
		return s.getDatabase(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName)
	case http.MethodDelete:
		return s.deleteDatabase(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CosmosDBService) handleContainerRequest(ctx *service.RequestContext, route cosmosRoute) (*service.Response, error) {
	if route.ContainerName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listContainers(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateContainer(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName, route.ContainerName, ctx.Body)
	case http.MethodGet:
		return s.getContainer(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName, route.ContainerName)
	case http.MethodDelete:
		return s.deleteContainer(route.SubscriptionID, route.ResourceGroup, route.AccountName, route.DatabaseName, route.ContainerName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CosmosDBService) createOrUpdateAccount(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["documentEndpoint"]; !ok {
		input.Properties["documentEndpoint"] = "https://" + name + ".documents.azure.com:443/"
	}

	account := DatabaseAccount{
		ID:         accountID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.DocumentDB/databaseAccounts",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := accountKey(subscriptionID, resourceGroup, name)
	_, existed := s.accounts[key]
	s.accounts[key] = account
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, account)
}

func (s *CosmosDBService) getAccount(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	account, ok := s.accounts[accountKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos DB account %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, account)
}

func (s *CosmosDBService) listAccounts(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"
	s.mu.RLock()
	values := make([]DatabaseAccount, 0)
	for key, account := range s.accounts {
		if strings.HasPrefix(key, prefix) {
			values = append(values, account)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CosmosDBService) deleteAccount(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := accountKey(subscriptionID, resourceGroup, name)
	if _, ok := s.accounts[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos DB account %q could not be found.", name))
	}
	delete(s.accounts, key)
	prefix := key + "/"
	for databaseKey := range s.databases {
		if strings.HasPrefix(databaseKey, prefix) {
			delete(s.databases, databaseKey)
		}
	}
	for containerKey := range s.containers {
		if strings.HasPrefix(containerKey, prefix) {
			delete(s.containers, containerKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *CosmosDBService) createOrUpdateDatabase(subscriptionID, resourceGroup, accountName, databaseName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
		Options    map[string]any `json:"options"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.accounts[accountKey(subscriptionID, resourceGroup, accountName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos DB account %q could not be found.", accountName))
	}
	database := SQLDatabase{
		ID:         databaseID(subscriptionID, resourceGroup, accountName, databaseName),
		Name:       accountName + "/" + databaseName,
		Type:       "Microsoft.DocumentDB/databaseAccounts/sqlDatabases",
		Properties: input.Properties,
		Options:    input.Options,
	}
	key := databaseKey(subscriptionID, resourceGroup, accountName, databaseName)
	_, existed := s.databases[key]
	s.databases[key] = database
	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, database)
}

func (s *CosmosDBService) getDatabase(subscriptionID, resourceGroup, accountName, databaseName string) (*service.Response, error) {
	s.mu.RLock()
	database, ok := s.databases[databaseKey(subscriptionID, resourceGroup, accountName, databaseName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL database %q could not be found.", databaseName))
	}
	return azurearm.JSONResponse(http.StatusOK, database)
}

func (s *CosmosDBService) listDatabases(subscriptionID, resourceGroup, accountName string) (*service.Response, error) {
	parentKey := accountKey(subscriptionID, resourceGroup, accountName)
	s.mu.RLock()
	_, accountExists := s.accounts[parentKey]
	values := make([]SQLDatabase, 0)
	prefix := parentKey + "/"
	for key, database := range s.databases {
		if strings.HasPrefix(key, prefix) {
			values = append(values, database)
		}
	}
	s.mu.RUnlock()
	if !accountExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos DB account %q could not be found.", accountName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CosmosDBService) deleteDatabase(subscriptionID, resourceGroup, accountName, databaseName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := databaseKey(subscriptionID, resourceGroup, accountName, databaseName)
	if _, ok := s.databases[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL database %q could not be found.", databaseName))
	}
	delete(s.databases, key)
	prefix := key + "/"
	for containerKey := range s.containers {
		if strings.HasPrefix(containerKey, prefix) {
			delete(s.containers, containerKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *CosmosDBService) createOrUpdateContainer(subscriptionID, resourceGroup, accountName, databaseName, containerName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
		Options    map[string]any `json:"options"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.databases[databaseKey(subscriptionID, resourceGroup, accountName, databaseName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL database %q could not be found.", databaseName))
	}
	container := SQLContainer{
		ID:         containerID(subscriptionID, resourceGroup, accountName, databaseName, containerName),
		Name:       accountName + "/" + databaseName + "/" + containerName,
		Type:       "Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers",
		Properties: input.Properties,
		Options:    input.Options,
	}
	key := containerKey(subscriptionID, resourceGroup, accountName, databaseName, containerName)
	_, existed := s.containers[key]
	s.containers[key] = container
	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, container)
}

func (s *CosmosDBService) getContainer(subscriptionID, resourceGroup, accountName, databaseName, containerName string) (*service.Response, error) {
	s.mu.RLock()
	container, ok := s.containers[containerKey(subscriptionID, resourceGroup, accountName, databaseName, containerName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL container %q could not be found.", containerName))
	}
	return azurearm.JSONResponse(http.StatusOK, container)
}

func (s *CosmosDBService) listContainers(subscriptionID, resourceGroup, accountName, databaseName string) (*service.Response, error) {
	parentKey := databaseKey(subscriptionID, resourceGroup, accountName, databaseName)
	s.mu.RLock()
	_, databaseExists := s.databases[parentKey]
	values := make([]SQLContainer, 0)
	prefix := parentKey + "/"
	for key, container := range s.containers {
		if strings.HasPrefix(key, prefix) {
			values = append(values, container)
		}
	}
	s.mu.RUnlock()
	if !databaseExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL database %q could not be found.", databaseName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CosmosDBService) deleteContainer(subscriptionID, resourceGroup, accountName, databaseName, containerName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := containerKey(subscriptionID, resourceGroup, accountName, databaseName, containerName)
	if _, ok := s.containers[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Cosmos SQL container %q could not be found.", containerName))
	}
	delete(s.containers, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

type cosmosRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	AccountName    string
	DatabaseType   string
	DatabaseName   string
	ContainerType  string
	ContainerName  string
}

func parseRoute(escapedPath string) (cosmosRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.DocumentDB") {
		return cosmosRoute{}, false
	}
	route := cosmosRoute{SubscriptionID: parts[1], ResourceGroup: parts[3], ResourceType: parts[6]}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.AccountName = parts[7]
		return route, true
	case 9:
		route.AccountName = parts[7]
		route.DatabaseType = parts[8]
		return route, true
	case 10:
		route.AccountName = parts[7]
		route.DatabaseType = parts[8]
		route.DatabaseName = parts[9]
		return route, true
	case 11:
		route.AccountName = parts[7]
		route.DatabaseType = parts[8]
		route.DatabaseName = parts[9]
		route.ContainerType = parts[10]
		return route, true
	case 12:
		route.AccountName = parts[7]
		route.DatabaseType = parts[8]
		route.DatabaseName = parts[9]
		route.ContainerType = parts[10]
		route.ContainerName = parts[11]
		return route, true
	default:
		return cosmosRoute{}, false
	}
}

func accountID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DocumentDB/databaseAccounts/" + name
}

func databaseID(subscriptionID, resourceGroup, accountName, databaseName string) string {
	return accountID(subscriptionID, resourceGroup, accountName) + "/sqlDatabases/" + databaseName
}

func containerID(subscriptionID, resourceGroup, accountName, databaseName, containerName string) string {
	return databaseID(subscriptionID, resourceGroup, accountName, databaseName) + "/containers/" + containerName
}

func accountKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func databaseKey(subscriptionID, resourceGroup, accountName, databaseName string) string {
	return accountKey(subscriptionID, resourceGroup, accountName) + "/" + strings.ToLower(databaseName)
}

func containerKey(subscriptionID, resourceGroup, accountName, databaseName, containerName string) string {
	return databaseKey(subscriptionID, resourceGroup, accountName, databaseName) + "/" + strings.ToLower(containerName)
}

func splitPath(escapedPath string) []string {
	trimmed := strings.Trim(escapedPath, "/")
	if trimmed == "" {
		return nil
	}
	rawParts := strings.Split(trimmed, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		if part == "" {
			continue
		}
		decoded, err := url.PathUnescape(part)
		if err != nil {
			decoded = part
		}
		parts = append(parts, decoded)
	}
	return parts
}

func stringifyTags(tags map[string]any) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = fmt.Sprint(value)
	}
	return out
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func splitNestedName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitTripleNestedName(name string) (string, string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
