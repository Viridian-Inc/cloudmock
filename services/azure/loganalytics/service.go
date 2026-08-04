package loganalytics

import (
	"crypto/sha256"
	"encoding/hex"
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

const (
	workspaceAPIVersion = "2025-02-01"
	queryAPIVersion     = "v1"
)

// LogAnalyticsService implements first-slice Azure Log Analytics control-plane APIs.
type LogAnalyticsService struct {
	mu         sync.RWMutex
	workspaces map[string]Workspace
}

func New() *LogAnalyticsService {
	return &LogAnalyticsService{workspaces: make(map[string]Workspace)}
}

func (s *LogAnalyticsService) Name() string { return "Microsoft.OperationalInsights" }

func (s *LogAnalyticsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateWorkspace", Method: http.MethodPut, IAMAction: "azure:Microsoft.OperationalInsights/workspaces/write"},
		{Name: "GetWorkspace", Method: http.MethodGet, IAMAction: "azure:Microsoft.OperationalInsights/workspaces/read"},
		{Name: "ListWorkspacesByResourceGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.OperationalInsights/workspaces/read"},
		{Name: "DeleteWorkspace", Method: http.MethodDelete, IAMAction: "azure:Microsoft.OperationalInsights/workspaces/delete"},
		{Name: "QueryWorkspace", Method: http.MethodPost, IAMAction: "azure:Microsoft.OperationalInsights/query/read"},
	}
}

func (s *LogAnalyticsService) HealthCheck() error { return nil }

func (s *LogAnalyticsService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.OperationalInsights/workspaces", APIVersion: workspaceAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.OperationalInsights/query", APIVersion: queryAPIVersion},
	}
}

func (s *LogAnalyticsService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.OperationalInsights/workspaces")
}

func (s *LogAnalyticsService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Log Analytics template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Log Analytics template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateWorkspace(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *LogAnalyticsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if route, ok := parseQueryRoute(ctx.RawRequest); ok {
		return s.handleWorkspaceQuery(ctx, route)
	}

	route, ok := parseWorkspaceRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Log Analytics route is not implemented.")
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listWorkspaces(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateWorkspace(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getWorkspace(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteWorkspace(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

type workspaceRoute struct {
	SubscriptionID string
	ResourceGroup  string
	Name           string
}

type queryRoute struct {
	WorkspaceID string
}

func (s *LogAnalyticsService) handleWorkspaceQuery(ctx *service.RequestContext, route queryRoute) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	var input struct {
		Query    string `json:"query"`
		Timespan string `json:"timespan"`
	}
	if len(ctx.Body) > 0 {
		if err := gojson.Unmarshal(ctx.Body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if strings.TrimSpace(input.Query) == "" {
		input.Query = ctx.RawRequest.URL.Query().Get("query")
	}
	if strings.TrimSpace(input.Query) == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadArgumentError", "The query field is required.")
	}
	return azurearm.JSONResponse(http.StatusOK, workspaceQueryResult(route.WorkspaceID, input.Query, input.Timespan))
}

func (s *LogAnalyticsService) createOrUpdateWorkspace(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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

	key := workspaceKey(subscriptionID, resourceGroup, name)
	id := workspaceID(subscriptionID, resourceGroup, name)

	s.mu.Lock()
	existing, existed := s.workspaces[key]
	properties := cloneMap(existing.Properties)
	for field, value := range input.Properties {
		properties[field] = value
	}
	normalizeWorkspaceProperties(id, existed, existing, properties)

	location := input.Location
	if location == "" && existed {
		location = existing.Location
	}
	if location == "" {
		location = "eastus"
	}

	workspace := Workspace{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.OperationalInsights/workspaces",
		Location:   location,
		Tags:       stringifyTags(input.Tags),
		Properties: properties,
	}
	s.workspaces[key] = workspace
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, workspace)
}

func (s *LogAnalyticsService) getWorkspace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	workspace, ok := s.workspaces[workspaceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Workspace %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, workspace)
}

func (s *LogAnalyticsService) listWorkspaces(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]Workspace, 0)
	for key, workspace := range s.workspaces {
		if strings.HasPrefix(key, prefix) {
			values = append(values, workspace)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *LogAnalyticsService) deleteWorkspace(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := workspaceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.workspaces[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.workspaces, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func parseWorkspaceRoute(escapedPath string) (workspaceRoute, bool) {
	parts := splitPath(escapedPath)
	providerIndex := -1
	for i, part := range parts {
		if strings.EqualFold(part, "providers") {
			providerIndex = i
			break
		}
	}
	if providerIndex != 4 || providerIndex+2 >= len(parts) ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[providerIndex+1], "Microsoft.OperationalInsights") ||
		!strings.EqualFold(parts[providerIndex+2], "workspaces") {
		return workspaceRoute{}, false
	}

	route := workspaceRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
	}
	switch len(parts) - providerIndex {
	case 3:
		return route, true
	case 4:
		route.Name = parts[providerIndex+3]
		return route, true
	default:
		return workspaceRoute{}, false
	}
}

func parseQueryRoute(r *http.Request) (queryRoute, bool) {
	host := normalizedHost(r)
	if host != "api.loganalytics.azure.com" && host != "api.loganalytics.io" {
		return queryRoute{}, false
	}
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) != 4 || !strings.EqualFold(parts[0], "v1") || !strings.EqualFold(parts[1], "workspaces") || !strings.EqualFold(parts[3], "query") {
		return queryRoute{}, false
	}
	return queryRoute{WorkspaceID: parts[2]}, true
}

func workspaceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.OperationalInsights/workspaces/" + name
}

func workspaceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func normalizeWorkspaceProperties(id string, existed bool, existing Workspace, properties map[string]any) {
	if properties == nil {
		return
	}
	if properties["customerId"] == nil {
		if existed && existing.Properties != nil && existing.Properties["customerId"] != nil {
			properties["customerId"] = existing.Properties["customerId"]
		} else {
			properties["customerId"] = deterministicUUID(id + "/customerId")
		}
	}
	setDefault(properties, "provisioningState", "Succeeded")
	setDefault(properties, "sku", map[string]any{"name": "PerGB2018"})
	setDefault(properties, "retentionInDays", 30)
	setDefault(properties, "publicNetworkAccessForIngestion", "Enabled")
	setDefault(properties, "publicNetworkAccessForQuery", "Enabled")
	setDefault(properties, "workspaceCapping", map[string]any{"dailyQuotaGb": -1, "dataIngestionStatus": "RespectQuota"})
	setDefault(properties, "createdDate", "2026-06-16T00:00:00Z")
	setDefault(properties, "modifiedDate", "2026-06-16T00:00:00Z")
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any)
	for key, value := range in {
		out[key] = value
	}
	return out
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

func setDefault(values map[string]any, key string, value any) {
	if _, ok := values[key]; !ok {
		values[key] = value
	}
}

func deterministicUUID(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	hexed := hex.EncodeToString(sum[:])
	return hexed[:8] + "-" + hexed[8:12] + "-" + hexed[12:16] + "-" + hexed[16:20] + "-" + hexed[20:32]
}

func workspaceQueryResult(workspaceID, query, timespan string) map[string]any {
	normalized := strings.ToLower(strings.TrimSpace(query))
	if strings.Contains(normalized, "summarize") && strings.Contains(normalized, "count()") && strings.Contains(normalized, "category") {
		return map[string]any{
			"tables": []map[string]any{
				{
					"name": "PrimaryResult",
					"columns": []map[string]string{
						{"name": "Category", "type": "string"},
						{"name": "count_", "type": "long"},
					},
					"rows": [][]any{
						{"Administrative", 1},
						{"Alert", 0},
					},
				},
			},
		}
	}
	if strings.Contains(normalized, "count") {
		return map[string]any{
			"tables": []map[string]any{
				{
					"name": "PrimaryResult",
					"columns": []map[string]string{
						{"name": "Count", "type": "long"},
					},
					"rows": [][]any{{1}},
				},
			},
		}
	}
	if strings.TrimSpace(timespan) == "" {
		timespan = "PT12H"
	}
	return map[string]any{
		"tables": []map[string]any{
			{
				"name": "PrimaryResult",
				"columns": []map[string]string{
					{"name": "TimeGenerated", "type": "datetime"},
					{"name": "WorkspaceId", "type": "string"},
					{"name": "TableName", "type": "string"},
					{"name": "Timespan", "type": "string"},
					{"name": "Message", "type": "string"},
				},
				"rows": [][]any{{
					"2026-06-16T00:00:00Z",
					workspaceID,
					tableNameFromQuery(query),
					timespan,
					"cloudmock synthetic log row",
				}},
			},
		},
	}
}

func tableNameFromQuery(query string) string {
	fields := strings.Fields(strings.TrimSpace(query))
	if len(fields) == 0 {
		return "Heartbeat"
	}
	return strings.Trim(fields[0], "|")
}

func normalizedHost(r *http.Request) string {
	host := strings.ToLower(r.Host)
	if host == "" {
		host = strings.ToLower(r.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	return host
}
