package containerapps

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

var containerAppsAPIVersions = []string{"2025-07-01"}

// ContainerAppsService implements first-slice Azure Container Apps ARM APIs.
type ContainerAppsService struct {
	mu                  sync.RWMutex
	managedEnvironments map[string]ManagedEnvironment
	containerApps       map[string]ContainerApp
	revisionStates      map[string]containerAppRevisionState
}

func New() *ContainerAppsService {
	return &ContainerAppsService{
		managedEnvironments: make(map[string]ManagedEnvironment),
		containerApps:       make(map[string]ContainerApp),
		revisionStates:      make(map[string]containerAppRevisionState),
	}
}

type containerAppRevisionState struct {
	Active       bool
	RunningState string
}

func (s *ContainerAppsService) Name() string { return "Microsoft.App" }

func (s *ContainerAppsService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateManagedEnvironment", Method: http.MethodPut, IAMAction: "azure:Microsoft.App/managedEnvironments/write"},
		{Name: "GetManagedEnvironment", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/managedEnvironments/read"},
		{Name: "ListManagedEnvironments", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/managedEnvironments/read"},
		{Name: "DeleteManagedEnvironment", Method: http.MethodDelete, IAMAction: "azure:Microsoft.App/managedEnvironments/delete"},
		{Name: "CreateOrUpdateContainerApp", Method: http.MethodPut, IAMAction: "azure:Microsoft.App/containerApps/write"},
		{Name: "GetContainerApp", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/containerApps/read"},
		{Name: "ListContainerApps", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/containerApps/read"},
		{Name: "DeleteContainerApp", Method: http.MethodDelete, IAMAction: "azure:Microsoft.App/containerApps/delete"},
		{Name: "ListContainerAppSecrets", Method: http.MethodPost, IAMAction: "azure:Microsoft.App/containerApps/listSecrets/action"},
		{Name: "ListContainerAppRevisions", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/containerApps/revisions/read"},
		{Name: "GetContainerAppRevision", Method: http.MethodGet, IAMAction: "azure:Microsoft.App/containerApps/revisions/read"},
		{Name: "ActivateContainerAppRevision", Method: http.MethodPost, IAMAction: "azure:Microsoft.App/containerApps/revisions/activate/action"},
		{Name: "DeactivateContainerAppRevision", Method: http.MethodPost, IAMAction: "azure:Microsoft.App/containerApps/revisions/deactivate/action"},
		{Name: "RestartContainerAppRevision", Method: http.MethodPost, IAMAction: "azure:Microsoft.App/containerApps/revisions/restart/action"},
	}
}

func (s *ContainerAppsService) HealthCheck() error { return nil }

func (s *ContainerAppsService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(containerAppsAPIVersions)*2)
	for _, apiVersion := range containerAppsAPIVersions {
		keys = append(keys,
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.App/containerApps", APIVersion: apiVersion},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.App/managedEnvironments", APIVersion: apiVersion},
		)
	}
	return keys
}

func (s *ContainerAppsService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.App/containerApps") ||
		strings.EqualFold(resourceType, "Microsoft.App/managedEnvironments")
}

func (s *ContainerAppsService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Container Apps template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Container Apps template resource is missing name")
	}
	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"identity":   resource["identity"],
		"kind":       resource["kind"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	var resp *service.Response
	if strings.EqualFold(stringValue(resource["type"]), "Microsoft.App/managedEnvironments") {
		resp, err = s.createOrUpdateManagedEnvironment(subscriptionID, resourceGroup, name, data)
	} else {
		resp, err = s.createOrUpdateContainerApp(subscriptionID, resourceGroup, name, data)
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

func (s *ContainerAppsService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Apps route is not implemented.")
	}
	switch strings.ToLower(route.ResourceType) {
	case "managedenvironments":
		return s.handleManagedEnvironmentRequest(ctx, route)
	case "containerapps":
		return s.handleContainerAppRequest(ctx, route)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Apps route is not implemented.")
	}
}

func (s *ContainerAppsService) handleManagedEnvironmentRequest(ctx *service.RequestContext, route containerAppsRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listManagedEnvironments(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateManagedEnvironment(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getManagedEnvironment(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteManagedEnvironment(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerAppsService) handleContainerAppRequest(ctx *service.RequestContext, route containerAppsRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listContainerApps(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.ActionName != "" {
		if strings.EqualFold(route.ActionName, "listSecrets") && ctx.RawRequest.Method == http.MethodPost {
			return s.listContainerAppSecrets(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Apps action is not implemented.")
	}
	if route.ChildType != "" {
		if strings.EqualFold(route.ChildType, "revisions") {
			if route.ChildName != "" && route.ChildAction != "" {
				if ctx.RawRequest.Method != http.MethodPost {
					return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
				}
				switch strings.ToLower(route.ChildAction) {
				case "activate":
					return s.activateContainerAppRevision(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
				case "deactivate":
					return s.deactivateContainerAppRevision(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
				case "restart":
					return s.restartContainerAppRevision(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
				default:
					return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Apps revision action is not implemented.")
				}
			}
			if route.ChildName == "" && ctx.RawRequest.Method == http.MethodGet {
				return s.listContainerAppRevisions(route.SubscriptionID, route.ResourceGroup, route.Name)
			}
			if route.ChildName != "" && ctx.RawRequest.Method == http.MethodGet {
				return s.getContainerAppRevision(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Apps child route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateContainerApp(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getContainerApp(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteContainerApp(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerAppsService) createOrUpdateManagedEnvironment(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
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
	normalizeManagedEnvironmentProperties(name, input.Location, input.Properties)

	environment := ManagedEnvironment{
		ID:         managedEnvironmentID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.App/managedEnvironments",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.managedEnvironments[key]
	s.managedEnvironments[key] = environment
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, environment)
}

func (s *ContainerAppsService) createOrUpdateContainerApp(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Identity   map[string]any `json:"identity"`
		Kind       string         `json:"kind"`
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
	template, _ := input.Properties["template"].(map[string]any)
	if len(asSlice(template["containers"])) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingContainerAppTemplate", "Container app properties.template.containers must contain at least one container.")
	}
	normalizeContainerAppProperties(name, input.Location, input.Properties)

	app := ContainerApp{
		ID:         containerAppID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.App/containerApps",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Kind:       input.Kind,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.containerApps[key]
	s.containerApps[key] = app
	stateKey := revisionStateKey(subscriptionID, resourceGroup, name, stringValue(app.Properties["latestRevisionName"]))
	if _, ok := s.revisionStates[stateKey]; !ok {
		s.revisionStates[stateKey] = defaultContainerAppRevisionState()
	}
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, app)
}

func (s *ContainerAppsService) getManagedEnvironment(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	environment, ok := s.managedEnvironments[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed environment %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, environment)
}

func (s *ContainerAppsService) getContainerApp(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, app)
}

func (s *ContainerAppsService) listContainerAppSecrets(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	configuration, _ := app.Properties["configuration"].(map[string]any)
	secrets := asSlice(configuration["secrets"])
	values := make([]map[string]any, 0, len(secrets))
	for _, secret := range secrets {
		if item, ok := secret.(map[string]any); ok {
			values = append(values, copyMap(item))
		}
	}
	sort.Slice(values, func(i, j int) bool { return stringValue(values[i]["name"]) < stringValue(values[j]["name"]) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerAppsService) listContainerAppRevisions(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	state := s.revisionStateLocked(subscriptionID, resourceGroup, name, projectedRevisionName(app))
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	revision := containerAppRevisionResponse(app, state)
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": []map[string]any{revision}})
}

func (s *ContainerAppsService) getContainerAppRevision(subscriptionID, resourceGroup, name, revisionName string) (*service.Response, error) {
	s.mu.RLock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	projectedName := projectedRevisionName(app)
	state := s.revisionStateLocked(subscriptionID, resourceGroup, name, projectedName)
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	if !strings.EqualFold(projectedName, revisionName) {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app revision %q could not be found.", revisionName))
	}
	revision := containerAppRevisionResponse(app, state)
	return azurearm.JSONResponse(http.StatusOK, revision)
}

func (s *ContainerAppsService) activateContainerAppRevision(subscriptionID, resourceGroup, name, revisionName string) (*service.Response, error) {
	return s.updateContainerAppRevisionState(subscriptionID, resourceGroup, name, revisionName, containerAppRevisionState{
		Active:       true,
		RunningState: "Running",
	})
}

func (s *ContainerAppsService) deactivateContainerAppRevision(subscriptionID, resourceGroup, name, revisionName string) (*service.Response, error) {
	return s.updateContainerAppRevisionState(subscriptionID, resourceGroup, name, revisionName, containerAppRevisionState{
		Active:       false,
		RunningState: "Stopped",
	})
}

func (s *ContainerAppsService) restartContainerAppRevision(subscriptionID, resourceGroup, name, revisionName string) (*service.Response, error) {
	s.mu.Lock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	projectedName := projectedRevisionName(app)
	if !strings.EqualFold(projectedName, revisionName) {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app revision %q could not be found.", revisionName))
	}
	state := s.revisionStateLocked(subscriptionID, resourceGroup, name, projectedName)
	state.RunningState = "Running"
	s.revisionStates[revisionStateKey(subscriptionID, resourceGroup, name, projectedName)] = state
	revision := containerAppRevisionResponse(app, state)
	s.mu.Unlock()
	return azurearm.JSONResponse(http.StatusOK, revision)
}

func (s *ContainerAppsService) updateContainerAppRevisionState(subscriptionID, resourceGroup, name, revisionName string, state containerAppRevisionState) (*service.Response, error) {
	s.mu.Lock()
	app, ok := s.containerApps[resourceKey(subscriptionID, resourceGroup, name)]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	projectedName := projectedRevisionName(app)
	if !strings.EqualFold(projectedName, revisionName) {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app revision %q could not be found.", revisionName))
	}
	s.revisionStates[revisionStateKey(subscriptionID, resourceGroup, name, projectedName)] = state
	revision := containerAppRevisionResponse(app, state)
	s.mu.Unlock()
	return azurearm.JSONResponse(http.StatusOK, revision)
}

func (s *ContainerAppsService) listManagedEnvironments(subscriptionID, resourceGroup string) (*service.Response, error) {
	s.mu.RLock()
	values := make([]ManagedEnvironment, 0)
	prefix := subscriptionResourcePrefix(subscriptionID, resourceGroup)
	for key, environment := range s.managedEnvironments {
		if strings.HasPrefix(key, prefix) {
			values = append(values, environment)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerAppsService) listContainerApps(subscriptionID, resourceGroup string) (*service.Response, error) {
	s.mu.RLock()
	values := make([]ContainerApp, 0)
	prefix := subscriptionResourcePrefix(subscriptionID, resourceGroup)
	for key, app := range s.containerApps {
		if strings.HasPrefix(key, prefix) {
			values = append(values, app)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerAppsService) deleteManagedEnvironment(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, ok := s.managedEnvironments[key]
	if ok {
		delete(s.managedEnvironments, key)
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Managed environment %q could not be found.", name))
	}
	return &service.Response{StatusCode: http.StatusAccepted, RawContentType: "application/json"}, nil
}

func (s *ContainerAppsService) deleteContainerApp(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, ok := s.containerApps[key]
	if ok {
		delete(s.containerApps, key)
		prefix := key + "/"
		for stateKey := range s.revisionStates {
			if strings.HasPrefix(stateKey, prefix) {
				delete(s.revisionStates, stateKey)
			}
		}
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container app %q could not be found.", name))
	}
	return &service.Response{StatusCode: http.StatusAccepted, RawContentType: "application/json"}, nil
}

func normalizeManagedEnvironmentProperties(name, location string, properties map[string]any) {
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["defaultDomain"]; !ok {
		properties["defaultDomain"] = name + "." + location + ".azurecontainerapps.io"
	}
	if _, ok := properties["staticIp"]; !ok {
		properties["staticIp"] = "10.0.0.4"
	}
}

func normalizeContainerAppProperties(name, location string, properties map[string]any) {
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["runningStatus"]; !ok {
		properties["runningStatus"] = "Running"
	}
	revisionName := name + "--000001"
	if _, ok := properties["latestRevisionName"]; !ok {
		properties["latestRevisionName"] = revisionName
	}
	if _, ok := properties["latestReadyRevisionName"]; !ok {
		properties["latestReadyRevisionName"] = revisionName
	}
	envName := resourceNameFromID(stringValue(properties["managedEnvironmentId"]))
	if envName == "" {
		envName = "default"
	}
	if _, ok := properties["environmentId"]; !ok {
		properties["environmentId"] = properties["managedEnvironmentId"]
	}
	if _, ok := properties["latestRevisionFqdn"]; !ok {
		properties["latestRevisionFqdn"] = revisionName + "." + envName + "." + location + ".azurecontainerapps.io"
	}
	if _, ok := properties["outboundIpAddresses"]; !ok {
		properties["outboundIpAddresses"] = []any{}
	}
}

type containerAppsRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	Name           string
	ActionName     string
	ChildType      string
	ChildName      string
	ChildAction    string
}

func parseRoute(escapedPath string) (containerAppsRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.App") {
		return containerAppsRoute{
			SubscriptionID: parts[1],
			ResourceType:   parts[4],
		}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.App") {
		return containerAppsRoute{}, false
	}
	route := containerAppsRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		if strings.EqualFold(parts[8], "revisions") {
			route.ChildType = parts[8]
		} else {
			route.ActionName = parts[8]
		}
		return route, true
	case 10:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	case 11:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.ChildAction = parts[10]
		return route, true
	default:
		return containerAppsRoute{}, false
	}
}

func containerAppRevisionResponse(app ContainerApp, state containerAppRevisionState) map[string]any {
	revisionName := projectedRevisionName(app)
	properties := map[string]any{
		"active":            state.Active,
		"fqdn":              stringValue(app.Properties["latestRevisionFqdn"]),
		"provisioningState": "Provisioned",
		"runningState":      state.RunningState,
		"template":          app.Properties["template"],
		"trafficWeight":     100,
	}
	return map[string]any{
		"id":         app.ID + "/revisions/" + revisionName,
		"name":       revisionName,
		"type":       "Microsoft.App/containerApps/revisions",
		"location":   app.Location,
		"properties": properties,
	}
}

func defaultContainerAppRevisionState() containerAppRevisionState {
	return containerAppRevisionState{Active: true, RunningState: "Running"}
}

func projectedRevisionName(app ContainerApp) string {
	revisionName := stringValue(app.Properties["latestRevisionName"])
	if revisionName == "" {
		revisionName = app.Name + "--000001"
	}
	return revisionName
}

func (s *ContainerAppsService) revisionStateLocked(subscriptionID, resourceGroup, name, revisionName string) containerAppRevisionState {
	state, ok := s.revisionStates[revisionStateKey(subscriptionID, resourceGroup, name, revisionName)]
	if !ok {
		return defaultContainerAppRevisionState()
	}
	if state.RunningState == "" {
		state.RunningState = "Running"
	}
	return state
}

func managedEnvironmentID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/managedEnvironments/" + name
}

func containerAppID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.App/containerApps/" + name
}

func resourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func revisionStateKey(subscriptionID, resourceGroup, name, revisionName string) string {
	return resourceKey(subscriptionID, resourceGroup, name) + "/" + strings.ToLower(revisionName)
}

func subscriptionResourcePrefix(subscriptionID, resourceGroup string) string {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}
	return prefix
}

func resourceNameFromID(id string) string {
	if id == "" {
		return ""
	}
	parts := splitPath(id)
	if len(parts) == 0 {
		return ""
	}
	return parts[len(parts)-1]
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

func asSlice(value any) []any {
	if typed, ok := value.([]any); ok {
		return typed
	}
	return nil
}

func copyMap(input map[string]any) map[string]any {
	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
