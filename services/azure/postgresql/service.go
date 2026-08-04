package postgresql

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

var postgresqlAPIVersions = []string{"2025-08-01", "2024-08-01"}

// PostgreSQLService implements first-slice Azure Database for PostgreSQL Flexible Server APIs.
type PostgreSQLService struct {
	mu            sync.RWMutex
	servers       map[string]Server
	databases     map[string]Database
	firewallRules map[string]FirewallRule
}

func New() *PostgreSQLService {
	return &PostgreSQLService{
		servers:       make(map[string]Server),
		databases:     make(map[string]Database),
		firewallRules: make(map[string]FirewallRule),
	}
}

func (s *PostgreSQLService) Name() string { return "Microsoft.DBforPostgreSQL" }

func (s *PostgreSQLService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateServer", Method: http.MethodPut, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/write"},
		{Name: "GetServer", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/read"},
		{Name: "ListServers", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/read"},
		{Name: "DeleteServer", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/delete"},
		{Name: "CreateDatabase", Method: http.MethodPut, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/databases/write"},
		{Name: "GetDatabase", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/databases/read"},
		{Name: "ListDatabases", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/databases/read"},
		{Name: "DeleteDatabase", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/databases/delete"},
		{Name: "CreateOrUpdateFirewallRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/firewallRules/write"},
		{Name: "GetFirewallRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/firewallRules/read"},
		{Name: "ListFirewallRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/firewallRules/read"},
		{Name: "DeleteFirewallRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.DBforPostgreSQL/flexibleServers/firewallRules/delete"},
	}
}

func (s *PostgreSQLService) HealthCheck() error { return nil }

func (s *PostgreSQLService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(postgresqlAPIVersions))
	for _, apiVersion := range postgresqlAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.DBforPostgreSQL/flexibleServers",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *PostgreSQLService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers") ||
		strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers/databases") ||
		strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules")
}

func (s *PostgreSQLService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported PostgreSQL template resource type %q", stringValue(resource["type"]))
	}
	resourceType := stringValue(resource["type"])
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("PostgreSQL template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"sku":        resource["sku"],
		"tags":       resource["tags"],
		"identity":   resource["identity"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	switch {
	case strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers"):
		resp, err = s.createOrUpdateServer(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers/databases"):
		serverName, databaseName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("PostgreSQL database template resource name must be {server}/{database}")
		}
		resp, err = s.createOrUpdateDatabase(subscriptionID, resourceGroup, serverName, databaseName, data)
	case strings.EqualFold(resourceType, "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules"):
		serverName, ruleName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("PostgreSQL firewall rule template resource name must be {server}/{rule}")
		}
		resp, err = s.createOrUpdateFirewallRule(subscriptionID, resourceGroup, serverName, ruleName, data)
	default:
		err = fmt.Errorf("unsupported PostgreSQL template resource type %q", resourceType)
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

func (s *PostgreSQLService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "flexibleServers") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The PostgreSQL route is not implemented.")
	}
	if route.ChildType != "" {
		switch {
		case strings.EqualFold(route.ChildType, "databases"):
			return s.handleDatabaseRequest(ctx, route)
		case strings.EqualFold(route.ChildType, "firewallRules"):
			return s.handleFirewallRuleRequest(ctx, route)
		default:
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The PostgreSQL route is not implemented.")
		}
	}
	return s.handleServerRequest(ctx, route)
}

func (s *PostgreSQLService) handleServerRequest(ctx *service.RequestContext, route postgresRoute) (*service.Response, error) {
	if route.ServerName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listServers(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateServer(route.SubscriptionID, route.ResourceGroup, route.ServerName, ctx.Body)
	case http.MethodGet:
		return s.getServer(route.SubscriptionID, route.ResourceGroup, route.ServerName)
	case http.MethodDelete:
		return s.deleteServer(route.SubscriptionID, route.ResourceGroup, route.ServerName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *PostgreSQLService) handleDatabaseRequest(ctx *service.RequestContext, route postgresRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listDatabases(route.SubscriptionID, route.ResourceGroup, route.ServerName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateDatabase(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getDatabase(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName)
	case http.MethodDelete:
		return s.deleteDatabase(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *PostgreSQLService) handleFirewallRuleRequest(ctx *service.RequestContext, route postgresRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listFirewallRules(route.SubscriptionID, route.ResourceGroup, route.ServerName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateFirewallRule(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getFirewallRule(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName)
	case http.MethodDelete:
		return s.deleteFirewallRule(route.SubscriptionID, route.ResourceGroup, route.ServerName, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *PostgreSQLService) createOrUpdateServer(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
		Identity   map[string]any `json:"identity"`
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
	if _, ok := input.Properties["state"]; !ok {
		input.Properties["state"] = "Ready"
	}
	if _, ok := input.Properties["fullyQualifiedDomainName"]; !ok {
		input.Properties["fullyQualifiedDomainName"] = name + ".postgres.database.azure.com"
	}

	server := Server{
		ID:         serverID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.DBforPostgreSQL/flexibleServers",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := serverKey(subscriptionID, resourceGroup, name)
	_, existed := s.servers[key]
	s.servers[key] = server
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, server)
}

func (s *PostgreSQLService) getServer(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	server, ok := s.servers[serverKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, server)
}

func (s *PostgreSQLService) listServers(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]Server, 0)
	for key, server := range s.servers {
		if strings.HasPrefix(key, prefix) {
			values = append(values, server)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *PostgreSQLService) deleteServer(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := serverKey(subscriptionID, resourceGroup, name)
	if _, ok := s.servers[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", name))
	}
	delete(s.servers, key)
	childPrefix := key + "/"
	for databaseKey := range s.databases {
		if strings.HasPrefix(databaseKey, childPrefix) {
			delete(s.databases, databaseKey)
		}
	}
	for firewallRuleKey := range s.firewallRules {
		if strings.HasPrefix(firewallRuleKey, childPrefix) {
			delete(s.firewallRules, firewallRuleKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *PostgreSQLService) createOrUpdateDatabase(subscriptionID, resourceGroup, serverName, databaseName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
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
	if _, ok := s.servers[serverKey(subscriptionID, resourceGroup, serverName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", serverName))
	}
	database := Database{
		ID:         databaseID(subscriptionID, resourceGroup, serverName, databaseName),
		Name:       serverName + "/" + databaseName,
		Type:       "Microsoft.DBforPostgreSQL/flexibleServers/databases",
		Properties: input.Properties,
	}
	key := databaseKey(subscriptionID, resourceGroup, serverName, databaseName)
	_, existed := s.databases[key]
	s.databases[key] = database

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, database)
}

func (s *PostgreSQLService) getDatabase(subscriptionID, resourceGroup, serverName, databaseName string) (*service.Response, error) {
	s.mu.RLock()
	database, ok := s.databases[databaseKey(subscriptionID, resourceGroup, serverName, databaseName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL database %q could not be found.", databaseName))
	}
	return azurearm.JSONResponse(http.StatusOK, database)
}

func (s *PostgreSQLService) listDatabases(subscriptionID, resourceGroup, serverName string) (*service.Response, error) {
	parentKey := serverKey(subscriptionID, resourceGroup, serverName)
	s.mu.RLock()
	_, serverExists := s.servers[parentKey]
	values := make([]Database, 0)
	prefix := parentKey + "/"
	for key, database := range s.databases {
		if strings.HasPrefix(key, prefix) {
			values = append(values, database)
		}
	}
	s.mu.RUnlock()
	if !serverExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", serverName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *PostgreSQLService) deleteDatabase(subscriptionID, resourceGroup, serverName, databaseName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := databaseKey(subscriptionID, resourceGroup, serverName, databaseName)
	if _, ok := s.databases[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL database %q could not be found.", databaseName))
	}
	delete(s.databases, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *PostgreSQLService) createOrUpdateFirewallRule(subscriptionID, resourceGroup, serverName, ruleName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.servers[serverKey(subscriptionID, resourceGroup, serverName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", serverName))
	}
	firewallRule := FirewallRule{
		ID:         firewallRuleID(subscriptionID, resourceGroup, serverName, ruleName),
		Name:       serverName + "/" + ruleName,
		Type:       "Microsoft.DBforPostgreSQL/flexibleServers/firewallRules",
		Properties: input.Properties,
	}
	key := firewallRuleKey(subscriptionID, resourceGroup, serverName, ruleName)
	_, existed := s.firewallRules[key]
	s.firewallRules[key] = firewallRule

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, firewallRule)
}

func (s *PostgreSQLService) getFirewallRule(subscriptionID, resourceGroup, serverName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	firewallRule, ok := s.firewallRules[firewallRuleKey(subscriptionID, resourceGroup, serverName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL firewall rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, firewallRule)
}

func (s *PostgreSQLService) listFirewallRules(subscriptionID, resourceGroup, serverName string) (*service.Response, error) {
	parentKey := serverKey(subscriptionID, resourceGroup, serverName)
	s.mu.RLock()
	_, serverExists := s.servers[parentKey]
	values := make([]FirewallRule, 0)
	prefix := parentKey + "/"
	for key, firewallRule := range s.firewallRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, firewallRule)
		}
	}
	s.mu.RUnlock()
	if !serverExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL server %q could not be found.", serverName))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *PostgreSQLService) deleteFirewallRule(subscriptionID, resourceGroup, serverName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := firewallRuleKey(subscriptionID, resourceGroup, serverName, ruleName)
	if _, ok := s.firewallRules[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("PostgreSQL firewall rule %q could not be found.", ruleName))
	}
	delete(s.firewallRules, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

type postgresRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	ServerName     string
	ChildType      string
	ChildName      string
}

func parseRoute(escapedPath string) (postgresRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.DBforPostgreSQL") {
		return postgresRoute{}, false
	}
	route := postgresRoute{SubscriptionID: parts[1], ResourceGroup: parts[3], ResourceType: parts[6]}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.ServerName = parts[7]
		return route, true
	case 9:
		route.ServerName = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.ServerName = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	default:
		return postgresRoute{}, false
	}
}

func serverID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.DBforPostgreSQL/flexibleServers/" + name
}

func databaseID(subscriptionID, resourceGroup, serverName, databaseName string) string {
	return serverID(subscriptionID, resourceGroup, serverName) + "/databases/" + databaseName
}

func firewallRuleID(subscriptionID, resourceGroup, serverName, ruleName string) string {
	return serverID(subscriptionID, resourceGroup, serverName) + "/firewallRules/" + ruleName
}

func serverKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func databaseKey(subscriptionID, resourceGroup, serverName, databaseName string) string {
	return serverKey(subscriptionID, resourceGroup, serverName) + "/" + strings.ToLower(databaseName)
}

func firewallRuleKey(subscriptionID, resourceGroup, serverName, ruleName string) string {
	return serverKey(subscriptionID, resourceGroup, serverName) + "/" + strings.ToLower(ruleName)
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
