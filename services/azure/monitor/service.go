package monitor

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

var (
	actionGroupAPIVersions       = []string{"2021-09-01", "2019-06-01"}
	metricAlertAPIVersions       = []string{"2024-03-01-preview", "2018-03-01"}
	diagnosticSettingAPIVersions = []string{"2021-05-01-preview"}
	metricsAPIVersions           = []string{"2023-10-01"}
	metricDefinitionAPIVersions  = []string{"2023-10-01"}
	activityLogAPIVersions       = []string{"2015-04-01"}
	componentAPIVersions         = []string{"2015-05-01"}
	queryAPIVersions             = []string{"v1"}
)

// MonitorService implements first-slice Azure Monitor control-plane APIs.
type MonitorService struct {
	mu                 sync.RWMutex
	actionGroups       map[string]ActionGroup
	metricAlerts       map[string]MetricAlert
	diagnosticSettings map[string]DiagnosticSetting
	components         map[string]ApplicationInsightsComponent
}

func New() *MonitorService {
	return &MonitorService{
		actionGroups:       make(map[string]ActionGroup),
		metricAlerts:       make(map[string]MetricAlert),
		diagnosticSettings: make(map[string]DiagnosticSetting),
		components:         make(map[string]ApplicationInsightsComponent),
	}
}

func (s *MonitorService) Name() string { return "Microsoft.Insights" }

func (s *MonitorService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateActionGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.Insights/actionGroups/write"},
		{Name: "GetActionGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/actionGroups/read"},
		{Name: "ListActionGroupsByResourceGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/actionGroups/read"},
		{Name: "DeleteActionGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Insights/actionGroups/delete"},
		{Name: "CreateOrUpdateMetricAlert", Method: http.MethodPut, IAMAction: "azure:Microsoft.Insights/metricAlerts/write"},
		{Name: "GetMetricAlert", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/metricAlerts/read"},
		{Name: "ListMetricAlertsByResourceGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/metricAlerts/read"},
		{Name: "DeleteMetricAlert", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Insights/metricAlerts/delete"},
		{Name: "CreateOrUpdateDiagnosticSetting", Method: http.MethodPut, IAMAction: "azure:Microsoft.Insights/diagnosticSettings/write"},
		{Name: "GetDiagnosticSetting", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/diagnosticSettings/read"},
		{Name: "ListDiagnosticSettings", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/diagnosticSettings/read"},
		{Name: "DeleteDiagnosticSetting", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Insights/diagnosticSettings/delete"},
		{Name: "ListMetrics", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/metrics/read"},
		{Name: "ListMetricDefinitions", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/metricDefinitions/read"},
		{Name: "ListActivityLogs", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/eventtypes/read"},
		{Name: "CreateOrUpdateComponent", Method: http.MethodPut, IAMAction: "azure:Microsoft.Insights/components/write"},
		{Name: "GetComponent", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/components/read"},
		{Name: "ListComponents", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/components/read"},
		{Name: "DeleteComponent", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Insights/components/delete"},
		{Name: "QueryGet", Method: http.MethodGet, IAMAction: "azure:Microsoft.Insights/query/read"},
	}
}

func (s *MonitorService) HealthCheck() error { return nil }

func (s *MonitorService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(actionGroupAPIVersions)+len(metricAlertAPIVersions)+len(diagnosticSettingAPIVersions)+len(metricsAPIVersions)+len(metricDefinitionAPIVersions)+len(activityLogAPIVersions)+len(componentAPIVersions)+len(queryAPIVersions))
	for _, version := range actionGroupAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/actionGroups",
			APIVersion: version,
		})
	}
	for _, version := range metricAlertAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/metricAlerts",
			APIVersion: version,
		})
	}
	for _, version := range diagnosticSettingAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/diagnosticSettings",
			APIVersion: version,
		})
	}
	for _, version := range metricsAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/metrics",
			APIVersion: version,
		})
	}
	for _, version := range metricDefinitionAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/metricDefinitions",
			APIVersion: version,
		})
	}
	for _, version := range activityLogAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/eventtypes",
			APIVersion: version,
		})
	}
	for _, version := range componentAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/components",
			APIVersion: version,
		})
	}
	for _, version := range queryAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Insights/query",
			APIVersion: version,
		})
	}
	return keys
}

func (s *MonitorService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.Insights/actionGroups") ||
		strings.EqualFold(resourceType, "Microsoft.Insights/metricAlerts") ||
		strings.EqualFold(resourceType, "Microsoft.Insights/diagnosticSettings") ||
		strings.EqualFold(resourceType, "Microsoft.Insights/components")
}

func (s *MonitorService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Monitor template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Monitor template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"kind":       resource["kind"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	switch resourceType := stringValue(resource["type"]); {
	case strings.EqualFold(resourceType, "Microsoft.Insights/actionGroups"):
		resp, err = s.createOrUpdateActionGroup(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Insights/metricAlerts"):
		resp, err = s.createOrUpdateMetricAlert(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Insights/diagnosticSettings"):
		scopeID := stringValue(resource["scope"])
		if scopeID == "" {
			return nil, fmt.Errorf("Monitor diagnostic setting template resource %q is missing scope", name)
		}
		resp, err = s.createOrUpdateDiagnosticSetting(scopeID, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Insights/components"):
		resp, err = s.createOrUpdateComponent(subscriptionID, resourceGroup, name, data)
	default:
		err = fmt.Errorf("unsupported Monitor template resource type %q", resourceType)
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

func (s *MonitorService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	if route, ok := parseQueryRoute(ctx.RawRequest.URL.EscapedPath()); ok {
		return s.handleQueryRequest(ctx, route)
	}
	if subscriptionID, ok := parseActivityLogsRoute(ctx.RawRequest.URL.EscapedPath()); ok {
		return s.handleActivityLogsRequest(ctx, subscriptionID)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Monitor route is not implemented.")
	}

	switch {
	case strings.EqualFold(route.ResourceType, "actionGroups"):
		return s.handleActionGroupRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "metricAlerts"):
		return s.handleMetricAlertRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "diagnosticSettings"):
		return s.handleDiagnosticSettingRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "metrics"):
		return s.handleMetricsRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "metricDefinitions"):
		return s.handleMetricDefinitionsRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "components"):
		return s.handleComponentRequest(ctx, route)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Monitor route is not implemented.")
	}
}

func (s *MonitorService) handleActionGroupRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listActionGroups(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateActionGroup(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getActionGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteActionGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *MonitorService) handleMetricAlertRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listMetricAlerts(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateMetricAlert(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getMetricAlert(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteMetricAlert(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *MonitorService) handleDiagnosticSettingRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listDiagnosticSettings(route.ScopeID)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateDiagnosticSetting(route.ScopeID, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getDiagnosticSetting(route.ScopeID, route.Name)
	case http.MethodDelete:
		return s.deleteDiagnosticSetting(route.ScopeID, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *MonitorService) handleMetricsRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name != "" || ctx.RawRequest.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	return s.listMetrics(route.ScopeID, ctx.RawRequest.URL.Query())
}

func (s *MonitorService) handleMetricDefinitionsRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name != "" || ctx.RawRequest.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	return s.listMetricDefinitions(route.ScopeID, ctx.RawRequest.URL.Query())
}

func (s *MonitorService) handleActivityLogsRequest(ctx *service.RequestContext, subscriptionID string) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	filter := strings.TrimSpace(ctx.RawRequest.URL.Query().Get("$filter"))
	if filter == "" {
		filter = strings.TrimSpace(ctx.RawRequest.URL.Query().Get("$Filter"))
	}
	if filter == "" || !strings.Contains(strings.ToLower(filter), "eventtimestamp ge") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidFilter", "The $filter query parameter is required and must include eventTimestamp ge.")
	}

	event := activityLogEvent(subscriptionID, filter)
	if fields := parseSelectFields(ctx.RawRequest.URL.Query()); len(fields) > 0 {
		event = selectActivityLogFields(event, fields)
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": []map[string]any{event}})
}

func (s *MonitorService) handleComponentRequest(ctx *service.RequestContext, route monitorRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listComponents(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateComponent(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getComponent(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteComponent(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *MonitorService) handleQueryRequest(ctx *service.RequestContext, route queryRoute) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	query := strings.TrimSpace(ctx.RawRequest.URL.Query().Get("query"))
	if query == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "BadArgumentError", "The query parameter is required.")
	}
	return azurearm.JSONResponse(http.StatusOK, applicationInsightsQueryResult(route.AppID, query))
}

func (s *MonitorService) createOrUpdateActionGroup(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
		input.Location = "Global"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"

	actionGroup := ActionGroup{
		ID:         actionGroupID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Insights/actionGroups",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.actionGroups[key]
	s.actionGroups[key] = actionGroup
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, actionGroup)
}

func (s *MonitorService) getActionGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	actionGroup, ok := s.actionGroups[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Action group %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, actionGroup)
}

func (s *MonitorService) listActionGroups(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]ActionGroup, 0)
	for key, actionGroup := range s.actionGroups {
		if strings.HasPrefix(key, prefix) {
			values = append(values, actionGroup)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *MonitorService) deleteActionGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.actionGroups[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Action group %q could not be found.", name))
	}
	delete(s.actionGroups, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *MonitorService) createOrUpdateMetricAlert(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
		input.Location = "global"
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"

	metricAlert := MetricAlert{
		ID:         metricAlertID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Insights/metricAlerts",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.metricAlerts[key]
	s.metricAlerts[key] = metricAlert
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, metricAlert)
}

func (s *MonitorService) getMetricAlert(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	metricAlert, ok := s.metricAlerts[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Metric alert %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, metricAlert)
}

func (s *MonitorService) listMetricAlerts(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]MetricAlert, 0)
	for key, metricAlert := range s.metricAlerts {
		if strings.HasPrefix(key, prefix) {
			values = append(values, metricAlert)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *MonitorService) deleteMetricAlert(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.metricAlerts[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Metric alert %q could not be found.", name))
	}
	delete(s.metricAlerts, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *MonitorService) createOrUpdateDiagnosticSetting(scopeID, name string, body []byte) (*service.Response, error) {
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

	setting := DiagnosticSetting{
		ID:         diagnosticSettingID(scopeID, name),
		Name:       name,
		Type:       "Microsoft.Insights/diagnosticSettings",
		Properties: input.Properties,
	}

	s.mu.Lock()
	s.diagnosticSettings[scopedResourceKey(scopeID, name)] = setting
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, setting)
}

func (s *MonitorService) getDiagnosticSetting(scopeID, name string) (*service.Response, error) {
	s.mu.RLock()
	setting, ok := s.diagnosticSettings[scopedResourceKey(scopeID, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Diagnostic setting %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, setting)
}

func (s *MonitorService) listDiagnosticSettings(scopeID string) (*service.Response, error) {
	prefix := normalizeScopeID(scopeID) + "/"

	s.mu.RLock()
	values := make([]DiagnosticSetting, 0)
	for key, setting := range s.diagnosticSettings {
		if strings.HasPrefix(key, prefix) {
			values = append(values, setting)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *MonitorService) deleteDiagnosticSetting(scopeID, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := scopedResourceKey(scopeID, name)
	if _, ok := s.diagnosticSettings[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Diagnostic setting %q could not be found.", name))
	}
	delete(s.diagnosticSettings, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *MonitorService) createOrUpdateComponent(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Kind       string         `json:"kind"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	key := resourceKey(subscriptionID, resourceGroup, name)
	id := componentID(subscriptionID, resourceGroup, name)

	s.mu.Lock()
	existing, existed := s.components[key]
	properties := make(map[string]any)
	if existed {
		for field, value := range existing.Properties {
			properties[field] = value
		}
	}
	for field, value := range input.Properties {
		properties[field] = value
	}
	normalizeComponentProperties(name, id, properties)

	location := input.Location
	if location == "" && existed {
		location = existing.Location
	}
	if location == "" {
		location = "eastus"
	}
	kind := input.Kind
	if kind == "" && existed {
		kind = existing.Kind
	}
	if kind == "" {
		kind = "web"
	}

	component := ApplicationInsightsComponent{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Insights/components",
		Kind:       kind,
		Location:   location,
		Tags:       stringifyTags(input.Tags),
		Properties: properties,
	}
	s.components[key] = component
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, component)
}

func (s *MonitorService) getComponent(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	component, ok := s.components[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Application Insights component %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, component)
}

func (s *MonitorService) listComponents(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := resourcePrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]ApplicationInsightsComponent, 0)
	for key, component := range s.components {
		if strings.HasPrefix(key, prefix) {
			values = append(values, component)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *MonitorService) deleteComponent(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.components[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.components, key)
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *MonitorService) listMetricDefinitions(scopeID string, query url.Values) (*service.Response, error) {
	namespace := strings.TrimSpace(query.Get("metricnamespace"))
	if namespace == "" {
		namespace = resourceNamespaceFromScope(scopeID)
	}
	if namespace == "" {
		namespace = "Microsoft.Insights"
	}

	definitions := []map[string]any{
		metricDefinition(scopeID, namespace, "Percentage CPU", "Percentage CPU", "Percent", "Average", "Average", "Maximum", "Minimum"),
		metricDefinition(scopeID, namespace, "Network In Total", "Network In Total", "Bytes", "Total", "Total", "Average", "Maximum"),
		metricDefinition(scopeID, namespace, "Network Out Total", "Network Out Total", "Bytes", "Total", "Total", "Average", "Maximum"),
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": definitions})
}

func (s *MonitorService) listMetrics(scopeID string, query url.Values) (*service.Response, error) {
	timespan := strings.TrimSpace(query.Get("timespan"))
	if timespan == "" {
		timespan = "2026-06-16T00:00:00Z/2026-06-16T01:00:00Z"
	}
	interval := strings.TrimSpace(query.Get("interval"))
	if interval == "" {
		interval = "PT1M"
	}
	namespace := strings.TrimSpace(query.Get("metricnamespace"))
	if namespace == "" {
		namespace = resourceNamespaceFromScope(scopeID)
	}

	metricNames := splitMetricNames(query.Get("metricnames"))
	if len(metricNames) == 0 {
		metricNames = []string{"Percentage CPU"}
	}

	values := make([]map[string]any, 0, len(metricNames))
	for _, name := range metricNames {
		values = append(values, metricValue(scopeID, namespace, name, timespan))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"cost":     0,
		"timespan": timespan,
		"interval": interval,
		"value":    values,
	})
}

type monitorRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ScopeID        string
	ResourceType   string
	Name           string
}

type queryRoute struct {
	APIVersion string
	AppID      string
}

func parseQueryRoute(escapedPath string) (queryRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) != 4 ||
		!strings.HasPrefix(strings.ToLower(parts[0]), "v") ||
		!strings.EqualFold(parts[1], "apps") ||
		!strings.EqualFold(parts[3], "query") {
		return queryRoute{}, false
	}
	return queryRoute{APIVersion: parts[0], AppID: parts[2]}, true
}

func parseRoute(escapedPath string) (monitorRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Insights") {
		return monitorRoute{
			SubscriptionID: parts[1],
			ScopeID:        "/subscriptions/" + parts[1],
			ResourceType:   parts[4],
		}, true
	}

	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") {
		return monitorRoute{}, false
	}

	providerIndex := -1
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.EqualFold(parts[i], "providers") && i+2 < len(parts) && strings.EqualFold(parts[i+1], "Microsoft.Insights") {
			providerIndex = i
			break
		}
	}
	if providerIndex < 0 || providerIndex+2 >= len(parts) {
		return monitorRoute{}, false
	}

	route := monitorRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ScopeID:        "/" + strings.Join(parts[:providerIndex], "/"),
		ResourceType:   parts[providerIndex+2],
	}
	switch len(parts) - providerIndex {
	case 3:
		return route, true
	case 4:
		route.Name = parts[providerIndex+3]
		return route, true
	default:
		return monitorRoute{}, false
	}
}

func parseActivityLogsRoute(escapedPath string) (string, bool) {
	parts := splitPath(escapedPath)
	if len(parts) != 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "providers") ||
		!strings.EqualFold(parts[3], "Microsoft.Insights") ||
		!strings.EqualFold(parts[4], "eventtypes") ||
		!strings.EqualFold(parts[5], "management") ||
		!strings.EqualFold(parts[6], "values") {
		return "", false
	}
	return parts[1], true
}

func actionGroupID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Insights/actionGroups/" + name
}

func metricAlertID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Insights/metricAlerts/" + name
}

func diagnosticSettingID(scopeID, name string) string {
	return strings.TrimRight(ensureLeadingSlash(scopeID), "/") + "/providers/Microsoft.Insights/diagnosticSettings/" + name
}

func componentID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Insights/components/" + name
}

func resourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func resourcePrefix(subscriptionID, resourceGroup string) string {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}
	return prefix
}

func scopedResourceKey(scopeID, name string) string {
	return normalizeScopeID(scopeID) + "/" + strings.ToLower(name)
}

func normalizeScopeID(scopeID string) string {
	return strings.ToLower(strings.TrimRight(ensureLeadingSlash(scopeID), "/"))
}

func ensureLeadingSlash(value string) string {
	if strings.HasPrefix(value, "/") {
		return value
	}
	return "/" + value
}

func splitMetricNames(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	names := make([]string, 0, len(parts))
	for _, part := range parts {
		if name := strings.TrimSpace(part); name != "" {
			names = append(names, name)
		}
	}
	return names
}

func resourceNamespaceFromScope(scopeID string) string {
	parts := splitPath(scopeID)
	for i := 0; i+2 < len(parts); i++ {
		if !strings.EqualFold(parts[i], "providers") {
			continue
		}
		resourceTypes := make([]string, 0)
		for j := i + 2; j < len(parts); j += 2 {
			resourceTypes = append(resourceTypes, parts[j])
		}
		if len(resourceTypes) == 0 {
			return parts[i+1]
		}
		return parts[i+1] + "/" + strings.Join(resourceTypes, "/")
	}
	return ""
}

func metricDefinition(scopeID, namespace, name, localizedName, unit, primaryAggregation string, supportedAggregations ...string) map[string]any {
	if len(supportedAggregations) == 0 {
		supportedAggregations = []string{primaryAggregation}
	}
	return map[string]any{
		"id":                        strings.TrimRight(ensureLeadingSlash(scopeID), "/") + "/providers/Microsoft.Insights/metricDefinitions/" + url.PathEscape(name),
		"resourceId":                ensureLeadingSlash(strings.TrimRight(scopeID, "/")),
		"namespace":                 namespace,
		"category":                  "Availability",
		"name":                      map[string]any{"value": name, "localizedValue": localizedName},
		"displayDescription":        localizedName,
		"isDimensionRequired":       false,
		"unit":                      unit,
		"primaryAggregationType":    primaryAggregation,
		"supportedAggregationTypes": supportedAggregations,
		"metricAvailabilities": []map[string]string{
			{"timeGrain": "PT1M", "retention": "P93D"},
			{"timeGrain": "PT5M", "retention": "P93D"},
			{"timeGrain": "PT1H", "retention": "P93D"},
			{"timeGrain": "P1D", "retention": "P93D"},
		},
		"dimensions": []any{},
	}
}

func metricValue(scopeID, namespace, name, timespan string) map[string]any {
	if namespace == "" {
		namespace = "Microsoft.Insights"
	}
	unit := "Count"
	average := float64(0)
	total := float64(0)
	switch strings.ToLower(name) {
	case "percentage cpu":
		unit = "Percent"
	case "network in total", "network out total":
		unit = "Bytes"
	}

	timestamp := firstTimespanTimestamp(timespan)
	return map[string]any{
		"id":                 strings.TrimRight(ensureLeadingSlash(scopeID), "/") + "/providers/Microsoft.Insights/metrics/" + url.PathEscape(name),
		"type":               "Microsoft.Insights/metrics",
		"name":               map[string]any{"value": name, "localizedValue": name},
		"displayDescription": name,
		"unit":               unit,
		"timeseries": []map[string]any{
			{
				"metadatavalues": []any{},
				"data": []map[string]any{
					{"timeStamp": timestamp, "average": average, "total": total, "minimum": average, "maximum": average},
				},
			},
		},
		"namespace": namespace,
	}
}

func firstTimespanTimestamp(timespan string) string {
	if before, _, ok := strings.Cut(timespan, "/"); ok && strings.TrimSpace(before) != "" {
		return strings.TrimSpace(before)
	}
	return "2026-06-16T00:00:00Z"
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

func normalizeComponentProperties(name, id string, properties map[string]any) {
	properties["ApplicationId"] = name
	setDefault(properties, "Application_Type", "web")
	setDefault(properties, "Flow_Type", "Bluefield")
	setDefault(properties, "Request_Source", "rest")
	setDefault(properties, "InstrumentationKey", deterministicUUID(id+"/instrumentationKey"))
	setDefault(properties, "AppId", deterministicUUID(id+"/appId"))
	setDefault(properties, "TenantId", deterministicUUID(id+"/tenantId"))
	setDefault(properties, "CreationDate", "2026-06-16T00:00:00Z")
	setDefault(properties, "HockeyAppId", "")
	setDefault(properties, "HockeyAppToken", "")
	setDefault(properties, "SamplingPercentage", 100)
	setDefault(properties, "RetentionInDays", 90)
	setDefault(properties, "DisableIpMasking", false)
	setDefault(properties, "ImmediatePurgeDataOn30Days", false)
	setDefault(properties, "IngestionMode", "ApplicationInsights")
	properties["ConnectionString"] = "InstrumentationKey=" + fmt.Sprint(properties["InstrumentationKey"])
	properties["provisioningState"] = "Succeeded"
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

func activityLogEvent(subscriptionID, filter string) map[string]any {
	resourceGroup := filterStringValue(filter, "resourceGroupName")
	if resourceGroup == "" {
		resourceGroup = "cloudmock"
	}
	resourceID := filterStringValue(filter, "resourceUri")
	if resourceID == "" {
		resourceID = "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Resources/deployments/cloudmock"
	}
	resourceProvider := filterStringValue(filter, "resourceProvider")
	if resourceProvider == "" {
		resourceProvider = "Microsoft.Resources"
	}
	correlationID := filterStringValue(filter, "correlationId")
	if correlationID == "" {
		correlationID = deterministicUUID(subscriptionID + "/" + resourceGroup + "/activity/correlation")
	}
	eventDataID := deterministicUUID(subscriptionID + "/" + resourceGroup + "/activity/event")
	operationName := resourceProvider + "/deployments/write"
	if !strings.EqualFold(resourceProvider, "Microsoft.Resources") {
		operationName = resourceProvider + "/write"
	}

	return map[string]any{
		"id":                  resourceID + "/events/" + eventDataID + "/ticks/638541504000000000",
		"eventDataId":         eventDataID,
		"correlationId":       correlationID,
		"operationId":         correlationID,
		"description":         "",
		"eventTimestamp":      "2026-06-16T00:00:00Z",
		"submissionTimestamp": "2026-06-16T00:00:05Z",
		"level":               "Informational",
		"resourceGroupName":   resourceGroup,
		"resourceId":          resourceID,
		"subscriptionId":      subscriptionID,
		"tenantId":            deterministicUUID(subscriptionID + "/tenant"),
		"caller":              "cloudmock@example.com",
		"claims": map[string]any{
			"aud": "https://management.core.windows.net/",
			"iss": "https://sts.windows.net/" + deterministicUUID(subscriptionID+"/tenant") + "/",
		},
		"authorization": map[string]any{
			"action": operationName,
			"role":   "Subscription Admin",
			"scope":  resourceID,
		},
		"httpRequest": map[string]any{
			"method":          "PUT",
			"clientIpAddress": "127.0.0.1",
			"clientRequestId": correlationID,
			"uri":             resourceID,
		},
		"eventName":            localizableString("EndRequest", "End request"),
		"operationName":        localizableString(operationName, operationName),
		"resourceProviderName": localizableString(resourceProvider, resourceProvider),
		"resourceType":         localizableString(resourceProvider+"/deployments", resourceProvider+"/deployments"),
		"status":               localizableString("Succeeded", "Succeeded"),
		"subStatus":            localizableString("Created", "Created (HTTP Status Code: 201)"),
		"properties": map[string]any{
			"statusCode": "Created",
		},
	}
}

func localizableString(value, localizedValue string) map[string]string {
	return map[string]string{"value": value, "localizedValue": localizedValue}
}

func filterStringValue(filter, field string) string {
	lower := strings.ToLower(filter)
	needle := strings.ToLower(field) + " eq '"
	start := strings.Index(lower, needle)
	if start < 0 {
		return ""
	}
	start += len(needle)
	end := strings.Index(filter[start:], "'")
	if end < 0 {
		return ""
	}
	return filter[start : start+end]
}

func parseSelectFields(query url.Values) map[string]bool {
	raw := query.Get("$select")
	if raw == "" {
		raw = query.Get("$Select")
	}
	if raw == "" {
		return nil
	}
	out := make(map[string]bool)
	for _, field := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			out[trimmed] = true
		}
	}
	return out
}

func selectActivityLogFields(event map[string]any, fields map[string]bool) map[string]any {
	selected := make(map[string]any, len(fields))
	for field := range fields {
		if value, ok := event[field]; ok {
			selected[field] = value
		}
	}
	return selected
}

func applicationInsightsQueryResult(appID, query string) map[string]any {
	if strings.Contains(strings.ToLower(query), "count") {
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

	itemType := "request"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(query)), "traces") {
		itemType = "trace"
	}

	return map[string]any{
		"tables": []map[string]any{
			{
				"name": "PrimaryResult",
				"columns": []map[string]string{
					{"name": "timestamp", "type": "datetime"},
					{"name": "id", "type": "string"},
					{"name": "name", "type": "string"},
					{"name": "url", "type": "string"},
					{"name": "success", "type": "string"},
					{"name": "resultCode", "type": "string"},
					{"name": "duration", "type": "real"},
					{"name": "appId", "type": "string"},
					{"name": "itemType", "type": "string"},
				},
				"rows": [][]any{
					{
						"2026-06-16T00:00:00Z",
						deterministicUUID(appID + "/" + query + "/row"),
						"GET /",
						"https://example.test/",
						"True",
						"200",
						0,
						appID,
						itemType,
					},
				},
			},
		},
	}
}
