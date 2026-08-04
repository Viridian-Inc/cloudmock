package containerinstance

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var containerInstanceAPIVersions = []string{"2025-09-01"}

// ContainerInstanceService implements first-slice Azure Container Instances ARM APIs.
type ContainerInstanceService struct {
	mu                             sync.RWMutex
	containerGroups                map[string]ContainerGroup
	containerGroupProfiles         map[string]ContainerGroupProfile
	containerGroupProfileRevisions map[string][]ContainerGroupProfile
}

func New() *ContainerInstanceService {
	return &ContainerInstanceService{
		containerGroups:                make(map[string]ContainerGroup),
		containerGroupProfiles:         make(map[string]ContainerGroupProfile),
		containerGroupProfileRevisions: make(map[string][]ContainerGroupProfile),
	}
}

func (s *ContainerInstanceService) Name() string { return "Microsoft.ContainerInstance" }

func (s *ContainerInstanceService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateContainerGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/write"},
		{Name: "GetContainerGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/read"},
		{Name: "ListContainerGroups", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/read"},
		{Name: "DeleteContainerGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/delete"},
		{Name: "StartContainerGroup", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/start/action"},
		{Name: "StopContainerGroup", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/stop/action"},
		{Name: "RestartContainerGroup", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/restart/action"},
		{Name: "ListContainerLogs", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/containers/logs/read"},
		{Name: "ExecuteContainerCommand", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/containers/exec/action"},
		{Name: "AttachContainer", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/containers/attach/action"},
		{Name: "GetOutboundNetworkDependenciesEndpoints", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroups/outboundNetworkDependenciesEndpoints/read"},
		{Name: "CreateOrUpdateContainerGroupProfile", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/write"},
		{Name: "GetContainerGroupProfile", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/read"},
		{Name: "ListContainerGroupProfiles", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/read"},
		{Name: "UpdateContainerGroupProfile", Method: http.MethodPatch, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/write"},
		{Name: "DeleteContainerGroupProfile", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/delete"},
		{Name: "ListContainerGroupProfileRevisions", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/revisions/read"},
		{Name: "GetContainerGroupProfileRevision", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/containerGroupProfiles/revisions/read"},
		{Name: "ListCachedImages", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/locations/cachedImages/read"},
		{Name: "ListCapabilities", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/locations/capabilities/read"},
		{Name: "ListUsage", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/locations/usages/read"},
		{Name: "ListOperations", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerInstance/operations/read"},
	}
}

func (s *ContainerInstanceService) HealthCheck() error { return nil }

func (s *ContainerInstanceService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(containerInstanceAPIVersions)*4)
	for _, apiVersion := range containerInstanceAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerInstance/containerGroups",
			APIVersion: apiVersion,
		})
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerInstance/containerGroupProfiles",
			APIVersion: apiVersion,
		})
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerInstance/locations",
			APIVersion: apiVersion,
		})
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerInstance/operations",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *ContainerInstanceService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.ContainerInstance/containerGroups") ||
		strings.EqualFold(resourceType, "Microsoft.ContainerInstance/containerGroupProfiles")
}

func (s *ContainerInstanceService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Container Instance template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Container Instance template resource is missing name")
	}
	body := map[string]any{
		"location":   resource["location"],
		"tags":       resource["tags"],
		"identity":   resource["identity"],
		"properties": resource["properties"],
		"zones":      resource["zones"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	var resp *service.Response
	if strings.EqualFold(stringValue(resource["type"]), "Microsoft.ContainerInstance/containerGroupProfiles") {
		resp, err = s.createOrUpdateContainerGroupProfile(subscriptionID, resourceGroup, name, data)
	} else {
		resp, err = s.createOrUpdateContainerGroup(subscriptionID, resourceGroup, name, data)
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

func (s *ContainerInstanceService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if parseOperationsRoute(ctx.RawRequest.URL.EscapedPath()) {
		if ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listOperations()
	}
	if route, ok := parseLocationRoute(ctx.RawRequest.URL.EscapedPath()); ok {
		return s.handleLocationRequest(ctx, route)
	}
	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Instance route is not implemented.")
	}
	if strings.EqualFold(route.ResourceType, "containerGroupProfiles") {
		return s.handleContainerGroupProfileRequest(ctx, route)
	}
	if !strings.EqualFold(route.ResourceType, "containerGroups") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Instance route is not implemented.")
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listContainerGroups(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.ChildType, "containers") && strings.EqualFold(route.ActionName, "logs") {
		return s.listContainerLogs(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.RawRequest)
	}
	if strings.EqualFold(route.ChildType, "containers") && strings.EqualFold(route.ActionName, "exec") {
		return s.executeContainerCommand(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.RawRequest, ctx.Body)
	}
	if strings.EqualFold(route.ChildType, "containers") && strings.EqualFold(route.ActionName, "attach") {
		return s.attachContainer(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.RawRequest)
	}
	if route.ActionName != "" {
		if strings.EqualFold(route.ActionName, "outboundNetworkDependenciesEndpoints") {
			return s.getOutboundNetworkDependenciesEndpoints(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.RawRequest)
		}
		return s.handleActionRequest(ctx, route)
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerInstanceService) handleContainerGroupProfileRequest(ctx *service.RequestContext, route containerInstanceRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listContainerGroupProfiles(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.ActionName, "revisions") {
		if ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listContainerGroupProfileRevisions(route.SubscriptionID, route.ResourceGroup, route.Name)
	}
	if strings.EqualFold(route.ChildType, "revisions") {
		if ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.getContainerGroupProfileRevision(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateContainerGroupProfile(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getContainerGroupProfile(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodPatch:
		return s.updateContainerGroupProfile(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodDelete:
		return s.deleteContainerGroupProfile(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerInstanceService) handleLocationRequest(ctx *service.RequestContext, route containerInstanceLocationRoute) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch {
	case strings.EqualFold(route.Operation, "cachedImages"):
		return s.listCachedImages(route.Location)
	case strings.EqualFold(route.Operation, "capabilities"):
		return s.listCapabilities(route.Location)
	case strings.EqualFold(route.Operation, "usages"):
		return s.listUsage(route.SubscriptionID, route.Location)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Instance location operation is not implemented.")
	}
}

func (s *ContainerInstanceService) handleActionRequest(ctx *service.RequestContext, route containerInstanceRoute) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch {
	case strings.EqualFold(route.ActionName, "start"):
		return s.startContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "stop"):
		return s.stopContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "restart"):
		return s.restartContainerGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Instance action is not implemented.")
	}
}

func (s *ContainerInstanceService) createOrUpdateContainerGroup(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
	if len(asSlice(input.Properties["containers"])) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingContainerGroupContainers", "Container group properties.containers must contain at least one container.")
	}
	normalizeContainerGroupProperties(name, input.Location, input.Properties)

	containerGroup := ContainerGroup{
		ID:         containerGroupID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.ContainerInstance/containerGroups",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := containerGroupKey(subscriptionID, resourceGroup, name)
	_, existed := s.containerGroups[key]
	s.containerGroups[key] = containerGroup
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, containerGroup)
}

func (s *ContainerInstanceService) getContainerGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	containerGroup, ok := s.containerGroups[containerGroupKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, containerGroup)
}

func (s *ContainerInstanceService) listContainerGroups(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]ContainerGroup, 0)
	for key, containerGroup := range s.containerGroups {
		if strings.HasPrefix(key, prefix) {
			values = append(values, containerGroup)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerInstanceService) createOrUpdateContainerGroupProfile(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
		Properties map[string]any `json:"properties"`
		Zones      []any          `json:"zones"`
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
	if len(asSlice(input.Properties["containers"])) == 0 {
		return azurearm.ErrorResponse(http.StatusBadRequest, "MissingContainerGroupProfileContainers", "Container group profile properties.containers must contain at least one container.")
	}

	key := containerGroupProfileKey(subscriptionID, resourceGroup, name)
	s.mu.Lock()
	_, existed := s.containerGroupProfiles[key]
	revision := len(s.containerGroupProfileRevisions[key]) + 1
	normalizeContainerGroupProfileProperties(input.Properties, revision)
	profile := ContainerGroupProfile{
		ID:         containerGroupProfileID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.ContainerInstance/containerGroupProfiles",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
		Zones:      cloneSlice(input.Zones),
	}
	s.containerGroupProfiles[key] = profile
	s.containerGroupProfileRevisions[key] = append(s.containerGroupProfileRevisions[key], cloneContainerGroupProfile(profile))
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, profile)
}

func (s *ContainerInstanceService) getContainerGroupProfile(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	profile, ok := s.containerGroupProfiles[containerGroupProfileKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group profile %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, profile)
}

func (s *ContainerInstanceService) listContainerGroupProfiles(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]ContainerGroupProfile, 0)
	for key, profile := range s.containerGroupProfiles {
		if strings.HasPrefix(key, prefix) {
			values = append(values, profile)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerInstanceService) updateContainerGroupProfile(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Tags map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	key := containerGroupProfileKey(subscriptionID, resourceGroup, name)
	profile, ok := s.containerGroupProfiles[key]
	if ok {
		profile.Tags = stringifyTags(input.Tags)
		s.containerGroupProfiles[key] = profile
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group profile %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, profile)
}

func (s *ContainerInstanceService) deleteContainerGroupProfile(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := containerGroupProfileKey(subscriptionID, resourceGroup, name)
	_, ok := s.containerGroupProfiles[key]
	if ok {
		delete(s.containerGroupProfiles, key)
		delete(s.containerGroupProfileRevisions, key)
	}
	s.mu.Unlock()
	if !ok {
		return &service.Response{
			StatusCode:     http.StatusNoContent,
			RawContentType: "application/json",
		}, nil
	}
	return &service.Response{
		StatusCode:     http.StatusOK,
		RawContentType: "application/json",
	}, nil
}

func (s *ContainerInstanceService) listContainerGroupProfileRevisions(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	revisions := cloneContainerGroupProfileSlice(s.containerGroupProfileRevisions[containerGroupProfileKey(subscriptionID, resourceGroup, name)])
	s.mu.RUnlock()
	if len(revisions) == 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group profile %q could not be found.", name))
	}
	sort.Slice(revisions, func(i, j int) bool {
		return revisionNumber(revisions[i].Properties["revision"]) < revisionNumber(revisions[j].Properties["revision"])
	})
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": revisions})
}

func (s *ContainerInstanceService) getContainerGroupProfileRevision(subscriptionID, resourceGroup, name, revision string) (*service.Response, error) {
	s.mu.RLock()
	revisions := cloneContainerGroupProfileSlice(s.containerGroupProfileRevisions[containerGroupProfileKey(subscriptionID, resourceGroup, name)])
	s.mu.RUnlock()
	for _, profile := range revisions {
		if strconv.Itoa(revisionNumber(profile.Properties["revision"])) == revision {
			return azurearm.JSONResponse(http.StatusOK, profile)
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group profile %q revision %q could not be found.", name, revision))
}

func (s *ContainerInstanceService) listOperations() (*service.Response, error) {
	seen := make(map[string]bool)
	values := make([]map[string]any, 0, len(s.Actions()))
	for _, action := range s.Actions() {
		name := strings.TrimPrefix(action.IAMAction, "azure:")
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		values = append(values, map[string]any{
			"name": name,
			"display": map[string]any{
				"provider":    "Microsoft Container Instance",
				"resource":    containerInstanceOperationResource(name),
				"operation":   action.Name,
				"description": action.Name + " operation.",
			},
			"origin": "User",
		})
	}
	sort.Slice(values, func(i, j int) bool {
		return stringValue(values[i]["name"]) < stringValue(values[j]["name"])
	})
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerInstanceService) listCachedImages(location string) (*service.Response, error) {
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"value": []map[string]any{
			{"osType": "Linux", "image": "ubuntu:22.04"},
			{"osType": "Linux", "image": "alpine:3.19"},
			{"osType": "Windows", "image": "mcr.microsoft.com/windows/nanoserver:ltsc2022"},
		},
	})
}

func (s *ContainerInstanceService) listCapabilities(location string) (*service.Response, error) {
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"capabilities": map[string]any{
					"maxCpu":        4,
					"maxGpuCount":   4,
					"maxMemoryInGB": 14,
				},
				"gpu":           "K80",
				"ipAddressType": "Public",
				"location":      location,
				"osType":        "Linux",
				"resourceType":  "containerGroups",
			},
			{
				"capabilities": map[string]any{
					"maxCpu":        4,
					"maxGpuCount":   0,
					"maxMemoryInGB": 14,
				},
				"gpu":           "None",
				"ipAddressType": "Public",
				"location":      location,
				"osType":        "Windows",
				"resourceType":  "containerGroups",
			},
		},
	})
}

func (s *ContainerInstanceService) listUsage(subscriptionID, location string) (*service.Response, error) {
	count := 0
	prefix := strings.ToLower(subscriptionID) + "/"

	s.mu.RLock()
	for key, containerGroup := range s.containerGroups {
		if strings.HasPrefix(key, prefix) && strings.EqualFold(containerGroup.Location, location) {
			count++
		}
	}
	s.mu.RUnlock()

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"name": map[string]any{
					"value":          "ContainerGroups",
					"localizedValue": "Container Groups",
				},
				"currentValue": count,
				"limit":        2000,
				"unit":         "Count",
			},
		},
	})
}

func (s *ContainerInstanceService) listContainerLogs(subscriptionID, resourceGroup, groupName, containerName string, req *http.Request) (*service.Response, error) {
	if req.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	s.mu.RLock()
	containerGroup, ok := s.containerGroups[containerGroupKey(subscriptionID, resourceGroup, groupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", groupName))
	}
	container, ok := findContainer(containerGroup.Properties, containerName)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container %q could not be found.", containerName))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{"content": containerLogsContent(containerGroup, container, req.URL.Query())})
}

func (s *ContainerInstanceService) executeContainerCommand(subscriptionID, resourceGroup, groupName, containerName string, req *http.Request, body []byte) (*service.Response, error) {
	if req.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	var input struct {
		Command      string         `json:"command"`
		TerminalSize map[string]any `json:"terminalSize"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.RLock()
	containerGroup, ok := s.containerGroups[containerGroupKey(subscriptionID, resourceGroup, groupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", groupName))
	}
	if _, ok := findContainer(containerGroup.Properties, containerName); !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container %q could not be found.", containerName))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"password":     "cloudmock-" + groupName + "-" + containerName + "-exec-token",
		"webSocketUri": containerExecWebSocketURI(subscriptionID, resourceGroup, groupName, containerName, input.Command, input.TerminalSize),
	})
}

func (s *ContainerInstanceService) attachContainer(subscriptionID, resourceGroup, groupName, containerName string, req *http.Request) (*service.Response, error) {
	if req.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	s.mu.RLock()
	containerGroup, ok := s.containerGroups[containerGroupKey(subscriptionID, resourceGroup, groupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", groupName))
	}
	if _, ok := findContainer(containerGroup.Properties, containerName); !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container %q could not be found.", containerName))
	}

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"password":     "cloudmock-" + groupName + "-" + containerName + "-attach-token",
		"webSocketUri": containerAttachWebSocketURI(subscriptionID, resourceGroup, groupName, containerName),
	})
}

func (s *ContainerInstanceService) getOutboundNetworkDependenciesEndpoints(subscriptionID, resourceGroup, name string, req *http.Request) (*service.Response, error) {
	if req.Method != http.MethodGet {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	s.mu.RLock()
	_, ok := s.containerGroups[containerGroupKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}

	return azurearm.JSONResponse(http.StatusOK, []string{})
}

func (s *ContainerInstanceService) deleteContainerGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := containerGroupKey(subscriptionID, resourceGroup, name)
	containerGroup, ok := s.containerGroups[key]
	if ok {
		delete(s.containerGroups, key)
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, containerGroup)
}

func (s *ContainerInstanceService) startContainerGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := containerGroupKey(subscriptionID, resourceGroup, name)
	containerGroup, ok := s.containerGroups[key]
	if ok {
		setContainerGroupRuntimeState(&containerGroup, true)
		s.containerGroups[key] = containerGroup
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}
	return containerInstanceAcceptedOperationResponse(subscriptionID, resourceGroup, name, "start"), nil
}

func (s *ContainerInstanceService) stopContainerGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := containerGroupKey(subscriptionID, resourceGroup, name)
	containerGroup, ok := s.containerGroups[key]
	if ok {
		setContainerGroupRuntimeState(&containerGroup, false)
		s.containerGroups[key] = containerGroup
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}
	return &service.Response{
		StatusCode:     http.StatusNoContent,
		RawContentType: "application/json",
	}, nil
}

func (s *ContainerInstanceService) restartContainerGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := containerGroupKey(subscriptionID, resourceGroup, name)
	containerGroup, ok := s.containerGroups[key]
	if ok {
		setContainerGroupRuntimeState(&containerGroup, true)
		s.containerGroups[key] = containerGroup
	}
	s.mu.Unlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container group %q could not be found.", name))
	}
	return &service.Response{
		StatusCode:     http.StatusNoContent,
		RawContentType: "application/json",
	}, nil
}

func containerInstanceAcceptedOperationResponse(subscriptionID, resourceGroup, name, operation string) *service.Response {
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location":    containerGroupID(subscriptionID, resourceGroup, name) + "/operationResults/" + strings.ToLower(operation),
			"Retry-After": "0",
		},
	}
}

func normalizeContainerGroupProperties(name, location string, properties map[string]any) {
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["osType"]; !ok {
		properties["osType"] = "Linux"
	}
	if _, ok := properties["restartPolicy"]; !ok {
		properties["restartPolicy"] = "Always"
	}
	if _, ok := properties["instanceView"]; !ok {
		properties["instanceView"] = map[string]any{
			"events": []any{},
			"state":  "Running",
		}
	}
	if ipAddress, ok := properties["ipAddress"].(map[string]any); ok {
		if _, ok := ipAddress["fqdn"]; !ok {
			label := stringValue(ipAddress["dnsNameLabel"])
			if label == "" {
				label = name
			}
			ipAddress["fqdn"] = label + "." + location + ".azurecontainer.io"
		}
	}
	for _, container := range asSlice(properties["containers"]) {
		containerMap, ok := container.(map[string]any)
		if !ok {
			continue
		}
		props, ok := containerMap["properties"].(map[string]any)
		if !ok {
			props = make(map[string]any)
			containerMap["properties"] = props
		}
		if _, ok := props["instanceView"]; ok {
			continue
		}
		props["instanceView"] = map[string]any{
			"currentState": map[string]any{
				"state":        "Running",
				"startTime":    deterministicStartTime(),
				"detailStatus": "Running",
			},
			"restartCount": 0,
			"events":       []any{},
		}
	}
}

func normalizeContainerGroupProfileProperties(properties map[string]any, revision int) {
	if _, ok := properties["osType"]; !ok {
		properties["osType"] = "Linux"
	}
	if _, ok := properties["restartPolicy"]; !ok {
		properties["restartPolicy"] = "Always"
	}
	properties["revision"] = revision
	registered := make([]int, 0, revision)
	for i := 1; i <= revision; i++ {
		registered = append(registered, i)
	}
	properties["registeredRevisions"] = registered
}

func setContainerGroupRuntimeState(containerGroup *ContainerGroup, running bool) {
	if containerGroup.Properties == nil {
		containerGroup.Properties = make(map[string]any)
	}
	normalizeContainerGroupProperties(containerGroup.Name, containerGroup.Location, containerGroup.Properties)

	groupView, ok := containerGroup.Properties["instanceView"].(map[string]any)
	if !ok {
		groupView = make(map[string]any)
		containerGroup.Properties["instanceView"] = groupView
	}
	if running {
		groupView["state"] = "Running"
	} else {
		groupView["state"] = "Stopped"
	}
	if _, ok := groupView["events"]; !ok {
		groupView["events"] = []any{}
	}

	for _, container := range asSlice(containerGroup.Properties["containers"]) {
		containerMap, ok := container.(map[string]any)
		if !ok {
			continue
		}
		props, ok := containerMap["properties"].(map[string]any)
		if !ok {
			props = make(map[string]any)
			containerMap["properties"] = props
		}
		instanceView, ok := props["instanceView"].(map[string]any)
		if !ok {
			instanceView = make(map[string]any)
			props["instanceView"] = instanceView
		}
		if running {
			instanceView["currentState"] = map[string]any{
				"state":        "Running",
				"startTime":    deterministicStartTime(),
				"detailStatus": "Running",
			}
		} else {
			instanceView["currentState"] = map[string]any{
				"state":        "Terminated",
				"finishTime":   deterministicStartTime(),
				"detailStatus": "Stopped",
			}
		}
		if _, ok := instanceView["restartCount"]; !ok {
			instanceView["restartCount"] = 0
		}
		if _, ok := instanceView["events"]; !ok {
			instanceView["events"] = []any{}
		}
	}
}

func cloneContainerGroupProfile(profile ContainerGroupProfile) ContainerGroupProfile {
	return ContainerGroupProfile{
		ID:         profile.ID,
		Name:       profile.Name,
		Type:       profile.Type,
		Location:   profile.Location,
		Tags:       cloneStringMap(profile.Tags),
		Properties: cloneMap(profile.Properties),
		Zones:      cloneSlice(profile.Zones),
	}
}

func cloneContainerGroupProfileSlice(values []ContainerGroupProfile) []ContainerGroupProfile {
	if len(values) == 0 {
		return nil
	}
	out := make([]ContainerGroupProfile, 0, len(values))
	for _, value := range values {
		out = append(out, cloneContainerGroupProfile(value))
	}
	return out
}

func findContainer(properties map[string]any, containerName string) (map[string]any, bool) {
	for _, container := range asSlice(properties["containers"]) {
		containerMap, ok := container.(map[string]any)
		if !ok {
			continue
		}
		if stringValue(containerMap["name"]) == containerName {
			return containerMap, true
		}
	}
	return nil, false
}

func containerLogsContent(containerGroup ContainerGroup, container map[string]any, query url.Values) string {
	props, _ := container["properties"].(map[string]any)
	image := stringValue(props["image"])
	if image == "" {
		image = "unknown"
	}
	state := "Running"
	if instanceView, ok := props["instanceView"].(map[string]any); ok {
		if currentState, ok := instanceView["currentState"].(map[string]any); ok {
			if current := stringValue(currentState["state"]); current != "" {
				state = current
			}
		}
	}
	lines := []string{
		"container " + stringValue(container["name"]) + " in group " + containerGroup.Name,
		"image " + image + " state " + state,
	}
	if tail, err := strconv.Atoi(strings.TrimSpace(query.Get("tail"))); err == nil && tail > 0 && tail < len(lines) {
		lines = lines[len(lines)-tail:]
	}
	if strings.EqualFold(query.Get("timestamps"), "true") {
		for i, line := range lines {
			lines[i] = deterministicStartTime() + " " + line
		}
	}
	return strings.Join(lines, "\n")
}

func containerExecWebSocketURI(subscriptionID, resourceGroup, groupName, containerName, command string, terminalSize map[string]any) string {
	query := url.Values{}
	if command != "" {
		query.Set("command", command)
	}
	if cols := intString(terminalSize["cols"]); cols != "" {
		query.Set("cols", cols)
	}
	if rows := intString(terminalSize["rows"]); rows != "" {
		query.Set("rows", rows)
	}
	base := "wss://management.azure.com" + containerGroupID(subscriptionID, resourceGroup, groupName) + "/containers/" + url.PathEscape(containerName) + "/exec/ws"
	if encoded := query.Encode(); encoded != "" {
		return base + "?" + encoded
	}
	return base
}

func containerAttachWebSocketURI(subscriptionID, resourceGroup, groupName, containerName string) string {
	return "wss://management.azure.com" + containerGroupID(subscriptionID, resourceGroup, groupName) + "/containers/" + url.PathEscape(containerName) + "/attach/ws"
}

type containerInstanceRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	Name           string
	ChildType      string
	ChildName      string
	ActionName     string
}

func parseOperationsRoute(escapedPath string) bool {
	parts := splitPath(escapedPath)
	return len(parts) == 3 &&
		strings.EqualFold(parts[0], "providers") &&
		strings.EqualFold(parts[1], "Microsoft.ContainerInstance") &&
		strings.EqualFold(parts[2], "operations")
}

type containerInstanceLocationRoute struct {
	SubscriptionID string
	Location       string
	Operation      string
}

func parseLocationRoute(escapedPath string) (containerInstanceLocationRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) != 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "providers") ||
		!strings.EqualFold(parts[3], "Microsoft.ContainerInstance") ||
		!strings.EqualFold(parts[4], "locations") {
		return containerInstanceLocationRoute{}, false
	}
	return containerInstanceLocationRoute{
		SubscriptionID: parts[1],
		Location:       parts[5],
		Operation:      parts[6],
	}, true
}

func parseRoute(escapedPath string) (containerInstanceRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ContainerInstance") &&
		(strings.EqualFold(parts[4], "containerGroups") || strings.EqualFold(parts[4], "containerGroupProfiles")) {
		return containerInstanceRoute{
			SubscriptionID: parts[1],
			ResourceType:   parts[4],
		}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ContainerInstance") {
		return containerInstanceRoute{}, false
	}
	route := containerInstanceRoute{
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
		route.ActionName = parts[8]
		return route, true
	case 10:
		if !strings.EqualFold(route.ResourceType, "containerGroupProfiles") || !strings.EqualFold(parts[8], "revisions") {
			return containerInstanceRoute{}, false
		}
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	case 11:
		if !strings.EqualFold(parts[8], "containers") || (!strings.EqualFold(parts[10], "logs") && !strings.EqualFold(parts[10], "exec") && !strings.EqualFold(parts[10], "attach")) {
			return containerInstanceRoute{}, false
		}
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.ActionName = parts[10]
		return route, true
	default:
		return containerInstanceRoute{}, false
	}
}

func containerGroupID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerInstance/containerGroups/" + name
}

func containerGroupProfileID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerInstance/containerGroupProfiles/" + name
}

func containerGroupKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func containerGroupProfileKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func containerInstanceOperationResource(operationName string) string {
	trimmed := strings.TrimPrefix(operationName, "Microsoft.ContainerInstance/")
	if trimmed == operationName {
		return "Container Instance"
	}
	parts := strings.Split(trimmed, "/")
	if len(parts) == 0 || parts[0] == "" {
		return "Container Instance"
	}
	switch parts[0] {
	case "containerGroups":
		return "Container Group"
	case "containerGroupProfiles":
		return "Container Group Profile"
	case "locations":
		return "Location"
	case "operations":
		return "Operation"
	default:
		return parts[0]
	}
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

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return nil
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return nil
	}
	data, err := gojson.Marshal(values)
	if err != nil {
		return values
	}
	var out map[string]any
	if err := gojson.Unmarshal(data, &out); err != nil {
		return values
	}
	return out
}

func cloneSlice(values []any) []any {
	if len(values) == 0 {
		return nil
	}
	data, err := gojson.Marshal(values)
	if err != nil {
		return append([]any(nil), values...)
	}
	var out []any
	if err := gojson.Unmarshal(data, &out); err != nil {
		return append([]any(nil), values...)
	}
	return out
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func intString(value any) string {
	switch typed := value.(type) {
	case float64:
		return strconv.Itoa(int(typed))
	case int:
		return strconv.Itoa(typed)
	default:
		return ""
	}
}

func revisionNumber(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int32:
		return int(typed)
	case int64:
		return int(typed)
	case float64:
		return int(typed)
	case string:
		parsed, _ := strconv.Atoi(typed)
		return parsed
	default:
		return 0
	}
}

func asSlice(value any) []any {
	switch typed := value.(type) {
	case []any:
		return typed
	default:
		return nil
	}
}

func deterministicStartTime() string {
	return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC).Format(time.RFC3339)
}
