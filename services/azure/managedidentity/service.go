package managedidentity

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

var managedIdentityAPIVersions = []string{"2023-01-31", "2018-11-30"}

// ManagedIdentityService implements Azure Managed Identity control-plane APIs.
type ManagedIdentityService struct {
	mu         sync.RWMutex
	identities map[string]UserAssignedIdentity
}

func New() *ManagedIdentityService {
	return &ManagedIdentityService{identities: make(map[string]UserAssignedIdentity)}
}

func (s *ManagedIdentityService) Name() string { return "Microsoft.ManagedIdentity" }

func (s *ManagedIdentityService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateUserAssignedIdentity", Method: http.MethodPut, IAMAction: "azure:Microsoft.ManagedIdentity/userAssignedIdentities/write"},
		{Name: "GetUserAssignedIdentity", Method: http.MethodGet, IAMAction: "azure:Microsoft.ManagedIdentity/userAssignedIdentities/read"},
		{Name: "ListUserAssignedIdentities", Method: http.MethodGet, IAMAction: "azure:Microsoft.ManagedIdentity/userAssignedIdentities/read"},
		{Name: "DeleteUserAssignedIdentity", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ManagedIdentity/userAssignedIdentities/delete"},
	}
}

func (s *ManagedIdentityService) HealthCheck() error { return nil }

func (s *ManagedIdentityService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(managedIdentityAPIVersions))
	for _, apiVersion := range managedIdentityAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ManagedIdentity/userAssignedIdentities",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *ManagedIdentityService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.ManagedIdentity/userAssignedIdentities")
}

func (s *ManagedIdentityService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Managed Identity template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Managed Identity template resource is missing name")
	}
	body := map[string]any{
		"location": resource["location"],
		"tags":     resource["tags"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateIdentity(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *ManagedIdentityService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "userAssignedIdentities") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Managed Identity route is not implemented.")
	}
	if route.IdentityName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listIdentities(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateIdentity(route.SubscriptionID, route.ResourceGroup, route.IdentityName, ctx.Body)
	case http.MethodGet:
		return s.getIdentity(route.SubscriptionID, route.ResourceGroup, route.IdentityName)
	case http.MethodDelete:
		return s.deleteIdentity(route.SubscriptionID, route.ResourceGroup, route.IdentityName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ManagedIdentityService) createOrUpdateIdentity(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location string         `json:"location"`
		Tags     map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}

	identity := UserAssignedIdentity{
		ID:          identityID(subscriptionID, resourceGroup, name),
		Name:        name,
		Type:        "Microsoft.ManagedIdentity/userAssignedIdentities",
		Location:    input.Location,
		Tags:        stringifyTags(input.Tags),
		ClientID:    "client-" + subscriptionID + "-" + resourceGroup + "-" + name,
		PrincipalID: "principal-" + subscriptionID + "-" + resourceGroup + "-" + name,
		TenantID:    "tenant-" + subscriptionID,
	}

	s.mu.Lock()
	key := identityKey(subscriptionID, resourceGroup, name)
	_, existed := s.identities[key]
	s.identities[key] = identity
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, identity)
}

func (s *ManagedIdentityService) getIdentity(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	identity, ok := s.identities[identityKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("User-assigned identity %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, identity)
}

func (s *ManagedIdentityService) listIdentities(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]UserAssignedIdentity, 0)
	for key, identity := range s.identities {
		if strings.HasPrefix(key, prefix) {
			values = append(values, identity)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ManagedIdentityService) deleteIdentity(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := identityKey(subscriptionID, resourceGroup, name)
	if _, ok := s.identities[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("User-assigned identity %q could not be found.", name))
	}
	delete(s.identities, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

type identityRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	IdentityName   string
}

func parseRoute(escapedPath string) (identityRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ManagedIdentity") {
		return identityRoute{}, false
	}
	route := identityRoute{SubscriptionID: parts[1], ResourceGroup: parts[3], ResourceType: parts[6]}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.IdentityName = parts[7]
		return route, true
	default:
		return identityRoute{}, false
	}
}

func identityID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ManagedIdentity/userAssignedIdentities/" + name
}

func identityKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
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
