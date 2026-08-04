package containerregistry

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

var containerRegistryAPIVersions = []string{"2025-11-01", "2023-07-01"}

const (
	containerRegistryDataPlaneAPIVersion = "2021-07-01"
	dockerDistributionAPIVersion         = "registry/2.0"
)

// ContainerRegistryService implements first-slice Azure Container Registry control-plane APIs.
type ContainerRegistryService struct {
	mu                 sync.RWMutex
	registries         map[string]Registry
	credentialVersions map[string]int
	repositories       map[string]map[string]*registryRepository
	uploadSessions     map[string]registryBlobUpload
	uploadSequence     int
}

func New() *ContainerRegistryService {
	return &ContainerRegistryService{
		registries:         make(map[string]Registry),
		credentialVersions: make(map[string]int),
		repositories:       make(map[string]map[string]*registryRepository),
		uploadSessions:     make(map[string]registryBlobUpload),
	}
}

func (s *ContainerRegistryService) Name() string { return "Microsoft.ContainerRegistry" }

func (s *ContainerRegistryService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateRegistry", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerRegistry/registries/write"},
		{Name: "UpdateRegistry", Method: http.MethodPatch, IAMAction: "azure:Microsoft.ContainerRegistry/registries/write"},
		{Name: "GetRegistry", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registries/read"},
		{Name: "ListRegistries", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registries/read"},
		{Name: "DeleteRegistry", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerRegistry/registries/delete"},
		{Name: "CheckNameAvailability", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerRegistry/checkNameAvailability/action"},
		{Name: "ListReplications", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registries/replications/read"},
		{Name: "ListUsages", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registries/listUsages/action"},
		{Name: "ImportImage", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerRegistry/registries/importImage/action"},
		{Name: "ListCredentials", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerRegistry/registries/listCredentials/action"},
		{Name: "RegenerateCredential", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerRegistry/registries/regenerateCredential/action"},
		{Name: "RegistryPing", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registry/pull"},
		{Name: "ListRepositories", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registry/pull"},
		{Name: "ListTags", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registry/pull"},
		{Name: "PutManifest", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerRegistry/registry/push"},
		{Name: "GetManifest", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registry/pull"},
		{Name: "DeleteManifest", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerRegistry/registry/delete"},
		{Name: "StartBlobUpload", Method: http.MethodPost, IAMAction: "azure:Microsoft.ContainerRegistry/registry/push"},
		{Name: "UploadBlobChunk", Method: http.MethodPatch, IAMAction: "azure:Microsoft.ContainerRegistry/registry/push"},
		{Name: "CompleteBlobUpload", Method: http.MethodPut, IAMAction: "azure:Microsoft.ContainerRegistry/registry/push"},
		{Name: "GetBlob", Method: http.MethodGet, IAMAction: "azure:Microsoft.ContainerRegistry/registry/pull"},
		{Name: "DeleteBlob", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerRegistry/registry/delete"},
		{Name: "CancelBlobUpload", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ContainerRegistry/registry/push"},
	}
}

func (s *ContainerRegistryService) HealthCheck() error { return nil }

func (s *ContainerRegistryService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(containerRegistryAPIVersions))
	for _, apiVersion := range containerRegistryAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ContainerRegistry/registries",
			APIVersion: apiVersion,
		})
	}
	keys = append(keys, routing.ServiceKey{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.ContainerRegistry/registry",
		APIVersion: containerRegistryDataPlaneAPIVersion,
	})
	return keys
}

func (s *ContainerRegistryService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.ContainerRegistry/registries")
}

func (s *ContainerRegistryService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Container Registry template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Container Registry template resource is missing name")
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
	resp, err := s.createRegistry(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ContainerRegistryService) TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool) {
	if !strings.EqualFold(operation, "listCredentials") || !s.SupportsTemplateResource(resource) {
		return nil, false
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, false
	}

	s.mu.RLock()
	if _, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]; !ok {
		s.mu.RUnlock()
		return nil, false
	}
	result := s.credentialsResponseLocked(subscriptionID, resourceGroup, name)
	s.mu.RUnlock()
	return result, true
}

func (s *ContainerRegistryService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if route, ok := parseDataPlaneRoute(ctx.RawRequest); ok {
		return s.handleDataPlaneRequest(ctx, route)
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "registries") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Registry route is not implemented.")
	}
	if route.ActionName != "" {
		return s.handleActionRequest(ctx, route)
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listRegistries(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createRegistry(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodPatch:
		return s.updateRegistry(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getRegistry(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteRegistry(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ContainerRegistryService) handleActionRequest(ctx *service.RequestContext, route containerRegistryRoute) (*service.Response, error) {
	switch {
	case strings.EqualFold(route.ActionName, "checkNameAvailability"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.checkNameAvailability(route.SubscriptionID, ctx.Body)
	case strings.EqualFold(route.ActionName, "replications"):
		if ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listReplications(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "listUsages"):
		if ctx.RawRequest.Method != http.MethodGet {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listUsages(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "importImage"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.importImage(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "listCredentials"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listCredentials(route.SubscriptionID, route.ResourceGroup, route.Name)
	case strings.EqualFold(route.ActionName, "regenerateCredential"):
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.regenerateCredential(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Container Registry action is not implemented.")
	}
}

func (s *ContainerRegistryService) checkNameAvailability(subscriptionID string, body []byte) (*service.Response, error) {
	var input struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if err := gojson.Unmarshal(body, &input); err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	name := strings.TrimSpace(input.Name)
	if !strings.EqualFold(input.Type, "Microsoft.ContainerRegistry/registries") || !isValidRegistryName(name) {
		return azurearm.JSONResponse(http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "Invalid",
			"message":       "The registry name is invalid.",
		})
	}

	s.mu.RLock()
	taken := false
	for key, registry := range s.registries {
		if !strings.HasPrefix(key, strings.ToLower(subscriptionID)+"/") {
			continue
		}
		if strings.EqualFold(registry.Name, name) {
			taken = true
			break
		}
	}
	s.mu.RUnlock()
	if taken {
		return azurearm.JSONResponse(http.StatusOK, map[string]any{
			"nameAvailable": false,
			"reason":        "AlreadyExists",
			"message":       "The registry " + name + " is already in use.",
		})
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"nameAvailable": true})
}

func (s *ContainerRegistryService) listReplications(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": []any{}})
}

func (s *ContainerRegistryService) listUsages(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"value": []map[string]any{
			{
				"name":         "Size",
				"limit":        int64(107374182400),
				"currentValue": int64(0),
				"unit":         "Bytes",
			},
		},
	})
}

func (s *ContainerRegistryService) importImage(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	return &service.Response{
		StatusCode:     http.StatusAccepted,
		RawContentType: "application/json",
	}, nil
}

func (s *ContainerRegistryService) createRegistry(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["loginServer"]; !ok {
		input.Properties["loginServer"] = name + ".azurecr.io"
	}
	if input.SKU == nil {
		input.SKU = map[string]any{"name": "Basic"}
	}
	normalizeSKU(input.SKU)

	registry := Registry{
		ID:         registryID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.ContainerRegistry/registries",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := registryKey(subscriptionID, resourceGroup, name)
	_, existed := s.registries[key]
	s.registries[key] = registry
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, registry)
}

func (s *ContainerRegistryService) updateRegistry(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
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

	s.mu.Lock()
	key := registryKey(subscriptionID, resourceGroup, name)
	registry, ok := s.registries[key]
	if !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	if input.Tags != nil {
		registry.Tags = stringifyTags(input.Tags)
	}
	if input.SKU != nil {
		normalizeSKU(input.SKU)
		registry.SKU = input.SKU
	}
	if input.Identity != nil {
		registry.Identity = input.Identity
	}
	if input.Properties != nil {
		if registry.Properties == nil {
			registry.Properties = make(map[string]any)
		}
		for key, value := range input.Properties {
			registry.Properties[key] = value
		}
		registry.Properties["provisioningState"] = "Succeeded"
		if _, ok := registry.Properties["loginServer"]; !ok {
			registry.Properties["loginServer"] = name + ".azurecr.io"
		}
	}
	s.registries[key] = registry
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, registry)
}

func (s *ContainerRegistryService) getRegistry(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	registry, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, registry)
}

func (s *ContainerRegistryService) listRegistries(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}

	s.mu.RLock()
	values := make([]Registry, 0)
	for key, registry := range s.registries {
		if strings.HasPrefix(key, prefix) {
			values = append(values, registry)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ContainerRegistryService) deleteRegistry(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := registryKey(subscriptionID, resourceGroup, name)
	if _, ok := s.registries[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	delete(s.registries, key)
	for credentialKey := range s.credentialVersions {
		if strings.HasPrefix(credentialKey, key+"/") {
			delete(s.credentialVersions, credentialKey)
		}
	}
	delete(s.repositories, strings.ToLower(name))
	for uploadKey := range s.uploadSessions {
		if strings.HasPrefix(uploadKey, strings.ToLower(name)+"/") {
			delete(s.uploadSessions, uploadKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *ContainerRegistryService) listCredentials(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]
	result := s.credentialsResponseLocked(subscriptionID, resourceGroup, name)
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, result)
}

func (s *ContainerRegistryService) credentialsResponseLocked(subscriptionID, resourceGroup, name string) map[string]any {
	return map[string]any{
		"username": name,
		"passwords": []any{
			map[string]any{"name": "password", "value": s.credentialValueLocked(subscriptionID, resourceGroup, name, "password")},
			map[string]any{"name": "password2", "value": s.credentialValueLocked(subscriptionID, resourceGroup, name, "password2")},
		},
	}
}

func (s *ContainerRegistryService) regenerateCredential(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Name string `json:"name"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Name == "" {
		input.Name = "password"
	}
	if !strings.EqualFold(input.Name, "password") && !strings.EqualFold(input.Name, "password2") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidCredentialName", "Credential name must be password or password2.")
	}
	input.Name = strings.ToLower(input.Name)

	s.mu.Lock()
	if _, ok := s.registries[registryKey(subscriptionID, resourceGroup, name)]; !ok {
		s.mu.Unlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Container registry %q could not be found.", name))
	}
	key := credentialKey(subscriptionID, resourceGroup, name, input.Name)
	currentVersion := s.credentialVersions[key]
	if currentVersion == 0 {
		currentVersion = 1
	}
	s.credentialVersions[key] = currentVersion + 1
	password := s.credentialValueLocked(subscriptionID, resourceGroup, name, input.Name)
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusOK, map[string]any{
		"username": name,
		"passwords": []map[string]string{
			{"name": input.Name, "value": password},
		},
	})
}

func (s *ContainerRegistryService) credentialValueLocked(subscriptionID, resourceGroup, registryName, credentialName string) string {
	version := s.credentialVersions[credentialKey(subscriptionID, resourceGroup, registryName, credentialName)]
	if version == 0 {
		version = 1
	}
	return "cm-acr-" + strings.ToLower(registryName) + "-" + strings.ToLower(credentialName) + "-" + fmt.Sprint(version)
}

type registryRepository struct {
	Manifests map[string]registryManifest
	Tags      map[string]string
	Blobs     map[string]registryBlob
}

type registryManifest struct {
	Digest    string
	MediaType string
	Content   []byte
}

type registryBlob struct {
	Digest  string
	Content []byte
}

type registryBlobUpload struct {
	ID           string
	RegistryName string
	Repository   string
	Content      []byte
}

type containerRegistryDataPlaneRoute struct {
	RegistryName string
	Parts        []string
}

func parseDataPlaneRoute(r *http.Request) (containerRegistryDataPlaneRoute, bool) {
	if r == nil || r.URL == nil {
		return containerRegistryDataPlaneRoute{}, false
	}
	host := strings.ToLower(r.Host)
	if host == "" {
		host = strings.ToLower(r.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	parts := splitPath(r.URL.EscapedPath())
	for _, suffix := range []string{".azurecr.io", ".azurecr.us", ".azurecr.cn", ".azurecr.de"} {
		if strings.HasSuffix(host, suffix) {
			name := strings.TrimSuffix(host, suffix)
			if name != "" && len(parts) > 0 && strings.EqualFold(parts[0], "v2") {
				return containerRegistryDataPlaneRoute{RegistryName: name, Parts: parts}, true
			}
		}
	}
	if len(parts) >= 2 && strings.HasSuffix(strings.ToLower(parts[0]), "-acr") && strings.EqualFold(parts[1], "v2") {
		name := strings.TrimSuffix(parts[0], "-acr")
		if name != "" {
			return containerRegistryDataPlaneRoute{RegistryName: name, Parts: parts[1:]}, true
		}
	}
	return containerRegistryDataPlaneRoute{}, false
}

func (s *ContainerRegistryService) handleDataPlaneRequest(ctx *service.RequestContext, route containerRegistryDataPlaneRoute) (*service.Response, error) {
	if len(route.Parts) == 0 || !strings.EqualFold(route.Parts[0], "v2") {
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", "The registry route is not implemented.")
	}

	switch {
	case len(route.Parts) == 1:
		if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
			return registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
		}
		return s.registryPing(route.RegistryName)
	case len(route.Parts) == 2 && strings.EqualFold(route.Parts[1], "_catalog"):
		if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
			return registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
		}
		return s.listRepositories(route.RegistryName)
	case len(route.Parts) >= 4 && strings.EqualFold(route.Parts[len(route.Parts)-2], "tags") && strings.EqualFold(route.Parts[len(route.Parts)-1], "list"):
		if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
			return registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
		}
		repository := strings.Join(route.Parts[1:len(route.Parts)-2], "/")
		return s.listTags(route.RegistryName, repository)
	default:
		if resp, handled, err := s.handleBlobDataPlaneRequest(ctx, route); handled {
			return resp, err
		}
		manifestIndex := -1
		for i := 1; i < len(route.Parts); i++ {
			if strings.EqualFold(route.Parts[i], "manifests") {
				manifestIndex = i
				break
			}
		}
		if manifestIndex < 2 || manifestIndex+2 != len(route.Parts) {
			return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", "The registry route is not implemented.")
		}
		repository := strings.Join(route.Parts[1:manifestIndex], "/")
		reference := route.Parts[manifestIndex+1]
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.putManifest(route.RegistryName, repository, reference, ctx.RawRequest.Header.Get("Content-Type"), ctx.Body)
		case http.MethodGet, http.MethodHead:
			return s.getManifest(route.RegistryName, repository, reference, ctx.RawRequest.Method == http.MethodHead)
		case http.MethodDelete:
			return s.deleteManifest(route.RegistryName, repository, reference)
		default:
			return registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
		}
	}
}

func (s *ContainerRegistryService) handleBlobDataPlaneRequest(ctx *service.RequestContext, route containerRegistryDataPlaneRoute) (*service.Response, bool, error) {
	blobIndex := -1
	for i := 1; i < len(route.Parts); i++ {
		if strings.EqualFold(route.Parts[i], "blobs") {
			blobIndex = i
			break
		}
	}
	if blobIndex < 2 || blobIndex+1 >= len(route.Parts) {
		return nil, false, nil
	}
	repository := strings.Join(route.Parts[1:blobIndex], "/")
	blobPart := route.Parts[blobIndex+1]
	if strings.EqualFold(blobPart, "uploads") {
		if len(route.Parts) == blobIndex+2 {
			if ctx.RawRequest.Method != http.MethodPost {
				resp, err := registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
				return resp, true, err
			}
			resp, err := s.startBlobUpload(route.RegistryName, repository)
			return resp, true, err
		}
		if len(route.Parts) != blobIndex+3 {
			resp, err := registryDataPlaneError(http.StatusNotFound, "BLOB_UPLOAD_UNKNOWN", "The blob upload could not be found.")
			return resp, true, err
		}
		uploadID := route.Parts[blobIndex+2]
		switch ctx.RawRequest.Method {
		case http.MethodPatch:
			resp, err := s.uploadBlobChunk(route.RegistryName, repository, uploadID, ctx.Body)
			return resp, true, err
		case http.MethodPut:
			resp, err := s.completeBlobUpload(route.RegistryName, repository, uploadID, ctx.RawRequest.URL.Query().Get("digest"), ctx.Body)
			return resp, true, err
		case http.MethodDelete:
			resp, err := s.cancelBlobUpload(route.RegistryName, uploadID)
			return resp, true, err
		default:
			resp, err := registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
			return resp, true, err
		}
	}
	if len(route.Parts) != blobIndex+2 {
		resp, err := registryDataPlaneError(http.StatusNotFound, "BLOB_UNKNOWN", "The requested blob could not be found.")
		return resp, true, err
	}
	digest := blobPart
	switch ctx.RawRequest.Method {
	case http.MethodGet, http.MethodHead:
		resp, err := s.getBlob(route.RegistryName, repository, digest, ctx.RawRequest.Method == http.MethodHead)
		return resp, true, err
	case http.MethodDelete:
		resp, err := s.deleteBlob(route.RegistryName, repository, digest)
		return resp, true, err
	default:
		resp, err := registryDataPlaneError(http.StatusMethodNotAllowed, "UNSUPPORTED", "The method is not allowed for this route.")
		return resp, true, err
	}
}

func (s *ContainerRegistryService) registryPing(registryName string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.findRegistryByNameLocked(registryName)
	s.mu.RUnlock()
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	return &service.Response{
		StatusCode: http.StatusOK,
		Headers:    registryDataPlaneHeaders(""),
	}, nil
}

func (s *ContainerRegistryService) listRepositories(registryName string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	registryRepos := s.repositories[strings.ToLower(registryName)]
	repositories := make([]string, 0, len(registryRepos))
	for name, repo := range registryRepos {
		if repo != nil {
			repositories = append(repositories, name)
		}
	}
	s.mu.RUnlock()
	sort.Strings(repositories)
	return registryDataPlaneJSONResponse(http.StatusOK, map[string]any{"repositories": repositories})
}

func (s *ContainerRegistryService) listTags(registryName, repository string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, false)
	tags := make([]string, 0)
	if repo != nil {
		tags = make([]string, 0, len(repo.Tags))
		for tag := range repo.Tags {
			tags = append(tags, tag)
		}
	}
	s.mu.RUnlock()
	sort.Strings(tags)
	return registryDataPlaneJSONResponse(http.StatusOK, map[string]any{"name": repository, "tags": tags})
}

func (s *ContainerRegistryService) putManifest(registryName, repository, reference, mediaType string, body []byte) (*service.Response, error) {
	if len(body) == 0 {
		return registryDataPlaneError(http.StatusBadRequest, "MANIFEST_INVALID", "The manifest content is required.")
	}
	if mediaType == "" {
		mediaType = "application/vnd.docker.distribution.manifest.v2+json"
	}
	sum := sha256.Sum256(body)
	digest := "sha256:" + hex.EncodeToString(sum[:])

	s.mu.Lock()
	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		s.mu.Unlock()
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, true)
	repo.Manifests[digest] = registryManifest{
		Digest:    digest,
		MediaType: mediaType,
		Content:   append([]byte(nil), body...),
	}
	if !isDigestReference(reference) {
		repo.Tags[reference] = digest
	}
	s.mu.Unlock()

	headers := registryDataPlaneHeaders(digest)
	headers["Location"] = "/v2/" + repository + "/manifests/" + reference
	return &service.Response{
		StatusCode: http.StatusCreated,
		Headers:    headers,
	}, nil
}

func (s *ContainerRegistryService) getManifest(registryName, repository, reference string, headOnly bool) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, false)
	if repo == nil {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "MANIFEST_UNKNOWN", "The requested manifest could not be found.")
	}
	digest := reference
	if taggedDigest, ok := repo.Tags[reference]; ok {
		digest = taggedDigest
	}
	manifest, ok := repo.Manifests[digest]
	s.mu.RUnlock()
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "MANIFEST_UNKNOWN", "The requested manifest could not be found.")
	}

	resp := &service.Response{
		StatusCode:     http.StatusOK,
		Headers:        registryDataPlaneHeaders(manifest.Digest),
		RawContentType: manifest.MediaType,
	}
	if !headOnly {
		resp.RawBody = append([]byte(nil), manifest.Content...)
	}
	return resp, nil
}

func (s *ContainerRegistryService) deleteManifest(registryName, repository, reference string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, false)
	if repo == nil {
		return registryDataPlaneError(http.StatusNotFound, "MANIFEST_UNKNOWN", "The requested manifest could not be found.")
	}
	digest := reference
	if taggedDigest, ok := repo.Tags[reference]; ok {
		digest = taggedDigest
		delete(repo.Tags, reference)
	} else if _, ok := repo.Manifests[digest]; ok {
		delete(repo.Manifests, digest)
		for tag, taggedDigest := range repo.Tags {
			if taggedDigest == digest {
				delete(repo.Tags, tag)
			}
		}
	} else {
		return registryDataPlaneError(http.StatusNotFound, "MANIFEST_UNKNOWN", "The requested manifest could not be found.")
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    registryDataPlaneHeaders(digest),
	}, nil
}

func (s *ContainerRegistryService) startBlobUpload(registryName, repository string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	s.repositoryLocked(registryName, repository, true)
	s.uploadSequence++
	uploadID := fmt.Sprintf("cm-upload-%d", s.uploadSequence)
	s.uploadSessions[blobUploadKey(registryName, uploadID)] = registryBlobUpload{
		ID:           uploadID,
		RegistryName: registryName,
		Repository:   repository,
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    blobUploadHeaders(repository, uploadID, 0),
	}, nil
}

func (s *ContainerRegistryService) uploadBlobChunk(registryName, repository, uploadID string, body []byte) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploadSessions[blobUploadKey(registryName, uploadID)]
	if !ok || upload.Repository != repository {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UPLOAD_UNKNOWN", "The blob upload could not be found.")
	}
	upload.Content = append(upload.Content, body...)
	s.uploadSessions[blobUploadKey(registryName, uploadID)] = upload
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    blobUploadHeaders(repository, uploadID, len(upload.Content)),
	}, nil
}

func (s *ContainerRegistryService) completeBlobUpload(registryName, repository, uploadID, digest string, body []byte) (*service.Response, error) {
	if digest == "" {
		return registryDataPlaneError(http.StatusBadRequest, "BLOB_UPLOAD_INVALID", "The digest query parameter is required.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	upload, ok := s.uploadSessions[blobUploadKey(registryName, uploadID)]
	if !ok || upload.Repository != repository {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UPLOAD_UNKNOWN", "The blob upload could not be found.")
	}
	content := append(append([]byte(nil), upload.Content...), body...)
	sum := sha256.Sum256(content)
	computed := "sha256:" + hex.EncodeToString(sum[:])
	if !strings.EqualFold(computed, digest) {
		return registryDataPlaneError(http.StatusBadRequest, "DIGEST_INVALID", "The provided digest does not match the uploaded blob content.")
	}
	repo := s.repositoryLocked(registryName, repository, true)
	repo.Blobs[digest] = registryBlob{
		Digest:  digest,
		Content: content,
	}
	delete(s.uploadSessions, blobUploadKey(registryName, uploadID))

	headers := registryDataPlaneHeaders(digest)
	headers["Location"] = "/v2/" + repository + "/blobs/" + digest
	headers["Content-Length"] = "0"
	return &service.Response{
		StatusCode: http.StatusCreated,
		Headers:    headers,
	}, nil
}

func (s *ContainerRegistryService) getBlob(registryName, repository, digest string, headOnly bool) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, false)
	if repo == nil {
		s.mu.RUnlock()
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UNKNOWN", "The requested blob could not be found.")
	}
	blob, ok := repo.Blobs[digest]
	s.mu.RUnlock()
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UNKNOWN", "The requested blob could not be found.")
	}
	headers := registryDataPlaneHeaders(blob.Digest)
	headers["Content-Length"] = fmt.Sprint(len(blob.Content))
	resp := &service.Response{
		StatusCode:     http.StatusOK,
		Headers:        headers,
		RawContentType: "application/octet-stream",
	}
	if !headOnly {
		resp.RawBody = append([]byte(nil), blob.Content...)
	}
	return resp, nil
}

func (s *ContainerRegistryService) deleteBlob(registryName, repository, digest string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, ok := s.findRegistryByNameLocked(registryName)
	if !ok {
		return registryDataPlaneError(http.StatusNotFound, "NAME_UNKNOWN", fmt.Sprintf("Container registry %q could not be found.", registryName))
	}
	repo := s.repositoryLocked(registryName, repository, false)
	if repo == nil {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UNKNOWN", "The requested blob could not be found.")
	}
	if _, ok := repo.Blobs[digest]; !ok {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UNKNOWN", "The requested blob could not be found.")
	}
	delete(repo.Blobs, digest)
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers:    registryDataPlaneHeaders(digest),
	}, nil
}

func (s *ContainerRegistryService) cancelBlobUpload(registryName, uploadID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := blobUploadKey(registryName, uploadID)
	if _, ok := s.uploadSessions[key]; !ok {
		return registryDataPlaneError(http.StatusNotFound, "BLOB_UPLOAD_UNKNOWN", "The blob upload could not be found.")
	}
	delete(s.uploadSessions, key)
	headers := registryDataPlaneHeaders("")
	headers["Content-Length"] = "0"
	return &service.Response{
		StatusCode: http.StatusNoContent,
		Headers:    headers,
	}, nil
}

func (s *ContainerRegistryService) findRegistryByNameLocked(name string) (Registry, bool) {
	for _, registry := range s.registries {
		if strings.EqualFold(registry.Name, name) {
			return registry, true
		}
	}
	return Registry{}, false
}

func (s *ContainerRegistryService) repositoryLocked(registryName, repository string, create bool) *registryRepository {
	registryKey := strings.ToLower(registryName)
	if s.repositories[registryKey] == nil {
		if !create {
			return nil
		}
		s.repositories[registryKey] = make(map[string]*registryRepository)
	}
	repo := s.repositories[registryKey][repository]
	if repo == nil && create {
		repo = &registryRepository{
			Manifests: make(map[string]registryManifest),
			Tags:      make(map[string]string),
			Blobs:     make(map[string]registryBlob),
		}
		s.repositories[registryKey][repository] = repo
	}
	if repo != nil && repo.Blobs == nil {
		repo.Blobs = make(map[string]registryBlob)
	}
	return repo
}

func registryDataPlaneJSONResponse(statusCode int, body any) (*service.Response, error) {
	resp, err := azurearm.JSONResponse(statusCode, body)
	if err != nil {
		return nil, err
	}
	resp.Headers = registryDataPlaneHeaders("")
	return resp, nil
}

func registryDataPlaneError(statusCode int, code, message string) (*service.Response, error) {
	return registryDataPlaneJSONResponse(statusCode, map[string]any{
		"errors": []map[string]any{
			{
				"code":    code,
				"message": message,
			},
		},
	})
}

func registryDataPlaneHeaders(digest string) map[string]string {
	headers := map[string]string{"Docker-Distribution-API-Version": dockerDistributionAPIVersion}
	if digest != "" {
		headers["Docker-Content-Digest"] = digest
	}
	return headers
}

func blobUploadHeaders(repository, uploadID string, size int) map[string]string {
	headers := registryDataPlaneHeaders("")
	headers["Docker-Upload-UUID"] = uploadID
	headers["Location"] = "/v2/" + repository + "/blobs/uploads/" + uploadID
	headers["Range"] = uploadRangeHeader(size)
	headers["Content-Length"] = "0"
	return headers
}

func uploadRangeHeader(size int) string {
	if size <= 0 {
		return "bytes=0-0"
	}
	return "bytes=0-" + fmt.Sprint(size-1)
}

func blobUploadKey(registryName, uploadID string) string {
	return strings.ToLower(registryName) + "/" + uploadID
}

func isDigestReference(reference string) bool {
	parts := strings.SplitN(reference, ":", 2)
	return len(parts) == 2 && parts[0] != "" && parts[1] != ""
}

type containerRegistryRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	Name           string
	ActionName     string
}

func parseRoute(escapedPath string) (containerRegistryRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ContainerRegistry") &&
		strings.EqualFold(parts[4], "checkNameAvailability") {
		return containerRegistryRoute{
			SubscriptionID: parts[1],
			ResourceType:   "registries",
			ActionName:     "checkNameAvailability",
		}, true
	}
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ContainerRegistry") &&
		strings.EqualFold(parts[4], "registries") {
		return containerRegistryRoute{
			SubscriptionID: parts[1],
			ResourceType:   "registries",
		}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ContainerRegistry") {
		return containerRegistryRoute{}, false
	}
	route := containerRegistryRoute{
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
	default:
		return containerRegistryRoute{}, false
	}
}

func registryID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerRegistry/registries/" + name
}

func registryKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func credentialKey(subscriptionID, resourceGroup, registryName, credentialName string) string {
	return registryKey(subscriptionID, resourceGroup, registryName) + "/" + strings.ToLower(credentialName)
}

func isValidRegistryName(name string) bool {
	if len(name) < 5 || len(name) > 50 {
		return false
	}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			continue
		}
		return false
	}
	return true
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

func normalizeSKU(sku map[string]any) {
	if len(sku) == 0 {
		return
	}
	if _, ok := sku["tier"]; ok {
		return
	}
	if name := stringValue(sku["name"]); name != "" {
		sku["tier"] = name
	}
}

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}
