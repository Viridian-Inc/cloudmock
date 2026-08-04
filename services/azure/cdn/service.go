package cdn

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

const cdnAPIVersion = "2025-04-15"

// CDNService implements first-slice Azure Front Door Standard/Premium profile APIs under Microsoft.Cdn.
type CDNService struct {
	mu            sync.RWMutex
	profiles      map[string]Profile
	endpoints     map[string]Endpoint
	originGroups  map[string]OriginGroup
	origins       map[string]Origin
	customDomains map[string]CustomDomain
}

func New() *CDNService {
	return &CDNService{
		profiles:      make(map[string]Profile),
		endpoints:     make(map[string]Endpoint),
		originGroups:  make(map[string]OriginGroup),
		origins:       make(map[string]Origin),
		customDomains: make(map[string]CustomDomain),
	}
}

func (s *CDNService) Name() string { return "Microsoft.Cdn" }

func (s *CDNService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cdn/profiles/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/read"},
		{Name: "List", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cdn/profiles/delete"},
		{Name: "CreateOrUpdateEndpoint", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/write"},
		{Name: "GetEndpoint", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/read"},
		{Name: "ListEndpoints", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/read"},
		{Name: "DeleteEndpoint", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/delete"},
		{Name: "CreateOrUpdateOriginGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/originGroups/write"},
		{Name: "GetOriginGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/originGroups/read"},
		{Name: "ListOriginGroups", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/originGroups/read"},
		{Name: "DeleteOriginGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/originGroups/delete"},
		{Name: "CreateOrUpdateOrigin", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/origins/write"},
		{Name: "GetOrigin", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/origins/read"},
		{Name: "ListOrigins", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/origins/read"},
		{Name: "DeleteOrigin", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/origins/delete"},
		{Name: "CreateOrUpdateCustomDomain", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/write"},
		{Name: "GetCustomDomain", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/read"},
		{Name: "ListCustomDomains", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/read"},
		{Name: "DeleteCustomDomain", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/delete"},
		{Name: "EnableCustomHttps", Method: http.MethodPost, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/enableCustomHttps/action"},
		{Name: "DisableCustomHttps", Method: http.MethodPost, IAMAction: "azure:Microsoft.Cdn/profiles/endpoints/customDomains/disableCustomHttps/action"},
	}
}

func (s *CDNService) HealthCheck() error { return nil }

func (s *CDNService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{{
		Provider:   routing.ProviderAzure,
		Service:    "Microsoft.Cdn/profiles",
		APIVersion: cdnAPIVersion,
	}}
}

func (s *CDNService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.Cdn/profiles") ||
		strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints") ||
		strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/originGroups") ||
		strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/origins") ||
		strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/customDomains")
}

func (s *CDNService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported CDN template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("CDN template resource is missing name")
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
	resourceType := stringValue(resource["type"])
	switch {
	case strings.EqualFold(resourceType, "Microsoft.Cdn/profiles"):
		resp, err = s.createOrUpdateProfile(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints"):
		profileName, endpointName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("CDN endpoint template resource name must be {profile}/{endpoint}")
		}
		resp, err = s.createOrUpdateEndpoint(subscriptionID, resourceGroup, profileName, endpointName, data)
	case strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/originGroups"):
		profileName, endpointName, originGroupName, ok := splitTripletName(name)
		if !ok {
			return nil, fmt.Errorf("CDN origin group template resource name must be {profile}/{endpoint}/{originGroup}")
		}
		resp, err = s.createOrUpdateOriginGroup(subscriptionID, resourceGroup, profileName, endpointName, originGroupName, data)
	case strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/origins"):
		profileName, endpointName, originName, ok := splitTripletName(name)
		if !ok {
			return nil, fmt.Errorf("CDN origin template resource name must be {profile}/{endpoint}/{origin}")
		}
		resp, err = s.createOrUpdateOrigin(subscriptionID, resourceGroup, profileName, endpointName, originName, data)
	case strings.EqualFold(resourceType, "Microsoft.Cdn/profiles/endpoints/customDomains"):
		profileName, endpointName, customDomainName, ok := splitTripletName(name)
		if !ok {
			return nil, fmt.Errorf("CDN custom domain template resource name must be {profile}/{endpoint}/{customDomain}")
		}
		resp, err = s.createOrUpdateCustomDomain(subscriptionID, resourceGroup, profileName, endpointName, customDomainName, data)
	default:
		err = fmt.Errorf("unsupported CDN template resource type %q", resourceType)
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

func (s *CDNService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "profiles") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.ChildType != "" {
		if strings.EqualFold(route.ChildType, "endpoints") {
			return s.handleEndpointRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.ProfileName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listProfiles(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateProfile(route.SubscriptionID, route.ResourceGroup, route.ProfileName, ctx.Body)
	case http.MethodGet:
		return s.getProfile(route.SubscriptionID, route.ResourceGroup, route.ProfileName)
	case http.MethodDelete:
		return s.deleteProfile(route.SubscriptionID, route.ResourceGroup, route.ProfileName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CDNService) createOrUpdateProfile(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        ProfileSKU     `json:"sku"`
		Tags       map[string]any `json:"tags"`
		Identity   map[string]any `json:"identity"`
		Properties struct {
			OriginResponseTimeoutSeconds int            `json:"originResponseTimeoutSeconds"`
			LogScrubbing                 map[string]any `json:"logScrubbing"`
			ExtendedProperties           map[string]any `json:"extendedProperties"`
		} `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "global"
	}
	if input.SKU.Name == "" {
		input.SKU.Name = "Standard_AzureFrontDoor"
	}
	timeout := input.Properties.OriginResponseTimeoutSeconds
	if timeout == 0 {
		timeout = 30
	}
	logScrubbing := input.Properties.LogScrubbing
	if logScrubbing == nil {
		logScrubbing = map[string]any{
			"state":          "Enabled",
			"scrubbingRules": []any{},
		}
	}

	profile := Profile{
		ID:       profileID(subscriptionID, resourceGroup, name),
		Name:     name,
		Type:     "Microsoft.Cdn/profiles",
		Location: input.Location,
		Tags:     stringifyTags(input.Tags),
		SKU:      input.SKU,
		Kind:     "frontdoor",
		Identity: input.Identity,
		Properties: ProfileProperties{
			OriginResponseTimeoutSeconds: timeout,
			LogScrubbing:                 logScrubbing,
			ExtendedProperties:           input.Properties.ExtendedProperties,
			FrontDoorID:                  deterministicFrontDoorID(subscriptionID, resourceGroup, name),
			ProvisioningState:            "Succeeded",
			ResourceState:                "Active",
		},
	}

	key := profileKey(subscriptionID, resourceGroup, name)
	s.mu.Lock()
	_, existed := s.profiles[key]
	s.profiles[key] = profile
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, profile)
}

func (s *CDNService) getProfile(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	profile, ok := s.profiles[profileKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN profile %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, profile)
}

func (s *CDNService) listProfiles(subscriptionID, resourceGroup string) (*service.Response, error) {
	subscriptionPrefix := strings.ToLower(subscriptionID) + "/"
	resourceGroupPrefix := profileKeyPrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]Profile, 0)
	for key, profile := range s.profiles {
		if resourceGroup != "" {
			if strings.HasPrefix(key, resourceGroupPrefix) {
				values = append(values, profile)
			}
			continue
		}
		if strings.HasPrefix(key, subscriptionPrefix) {
			values = append(values, profile)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].ID < values[j].ID
		}
		return values[i].Name < values[j].Name
	})
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CDNService) deleteProfile(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := profileKey(subscriptionID, resourceGroup, name)
	if _, ok := s.profiles[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.profiles, key)
	endpointPrefix := key + "/"
	for endpointKey := range s.endpoints {
		if strings.HasPrefix(endpointKey, endpointPrefix) {
			delete(s.endpoints, endpointKey)
		}
	}
	for originGroupKey := range s.originGroups {
		if strings.HasPrefix(originGroupKey, endpointPrefix) {
			delete(s.originGroups, originGroupKey)
		}
	}
	for originKey := range s.origins {
		if strings.HasPrefix(originKey, endpointPrefix) {
			delete(s.origins, originKey)
		}
	}
	for customDomainKey := range s.customDomains {
		if strings.HasPrefix(customDomainKey, endpointPrefix) {
			delete(s.customDomains, customDomainKey)
		}
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location": profileID(subscriptionID, resourceGroup, name) + "/operationResults/delete?api-version=" + cdnAPIVersion,
		},
	}, nil
}

func (s *CDNService) handleEndpointRequest(ctx *service.RequestContext, route cdnRoute) (*service.Response, error) {
	if route.ProfileName == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.GrandchildType != "" {
		if strings.EqualFold(route.GrandchildType, "originGroups") {
			return s.handleOriginGroupRequest(ctx, route)
		}
		if strings.EqualFold(route.GrandchildType, "origins") {
			return s.handleOriginRequest(ctx, route)
		}
		if strings.EqualFold(route.GrandchildType, "customDomains") {
			return s.handleCustomDomainRequest(ctx, route)
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.EndpointName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listEndpoints(route.SubscriptionID, route.ResourceGroup, route.ProfileName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateEndpoint(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, ctx.Body)
	case http.MethodGet:
		return s.getEndpoint(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName)
	case http.MethodDelete:
		return s.deleteEndpoint(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CDNService) createOrUpdateEndpoint(subscriptionID, resourceGroup, profileName, endpointName string, body []byte) (*service.Response, error) {
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

	s.mu.Lock()
	defer s.mu.Unlock()

	profile, profileExists := s.profiles[profileKey(subscriptionID, resourceGroup, profileName)]
	if !profileExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN profile %q could not be found.", profileName))
	}
	if input.Location == "" {
		input.Location = profile.Location
		if input.Location == "" {
			input.Location = "global"
		}
	}
	properties := cloneMap(input.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["hostName"]; !ok {
		properties["hostName"] = endpointName + ".azureedge.net"
	}
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["resourceState"]; !ok {
		properties["resourceState"] = "Running"
	}
	if _, ok := properties["isHttpAllowed"]; !ok {
		properties["isHttpAllowed"] = true
	}
	if _, ok := properties["isHttpsAllowed"]; !ok {
		properties["isHttpsAllowed"] = true
	}
	if _, ok := properties["queryStringCachingBehavior"]; !ok {
		properties["queryStringCachingBehavior"] = "NotSet"
	}
	if _, ok := properties["contentTypesToCompress"]; !ok {
		properties["contentTypesToCompress"] = []any{}
	}
	if _, ok := properties["geoFilters"]; !ok {
		properties["geoFilters"] = []any{}
	}
	if _, ok := properties["origins"]; !ok {
		properties["origins"] = []any{}
	}
	if _, ok := properties["originGroups"]; !ok {
		properties["originGroups"] = []any{}
	}
	if _, ok := properties["customDomains"]; !ok {
		properties["customDomains"] = []any{}
	}

	endpoint := Endpoint{
		ID:         endpointID(subscriptionID, resourceGroup, profileName, endpointName),
		Name:       endpointName,
		Type:       "Microsoft.Cdn/profiles/endpoints",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: properties,
	}

	key := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	_, existed := s.endpoints[key]
	s.endpoints[key] = endpoint

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, endpoint)
}

func (s *CDNService) getEndpoint(subscriptionID, resourceGroup, profileName, endpointName string) (*service.Response, error) {
	s.mu.RLock()
	endpoint, ok := s.endpoints[endpointKey(subscriptionID, resourceGroup, profileName, endpointName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	return azurearm.JSONResponse(http.StatusOK, endpoint)
}

func (s *CDNService) listEndpoints(subscriptionID, resourceGroup, profileName string) (*service.Response, error) {
	parentKey := profileKey(subscriptionID, resourceGroup, profileName)
	prefix := parentKey + "/"

	s.mu.RLock()
	_, parentExists := s.profiles[parentKey]
	values := make([]Endpoint, 0)
	for key, endpoint := range s.endpoints {
		if strings.HasPrefix(key, prefix) {
			values = append(values, endpoint)
		}
	}
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN profile %q could not be found.", profileName))
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].ID < values[j].ID
		}
		return values[i].Name < values[j].Name
	})
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CDNService) deleteEndpoint(subscriptionID, resourceGroup, profileName, endpointName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if _, ok := s.endpoints[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.endpoints, key)
	originGroupPrefix := key + "/"
	for originGroupKey := range s.originGroups {
		if strings.HasPrefix(originGroupKey, originGroupPrefix) {
			delete(s.originGroups, originGroupKey)
		}
	}
	for originKey := range s.origins {
		if strings.HasPrefix(originKey, originGroupPrefix) {
			delete(s.origins, originKey)
		}
	}
	for customDomainKey := range s.customDomains {
		if strings.HasPrefix(customDomainKey, originGroupPrefix) {
			delete(s.customDomains, customDomainKey)
		}
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location": endpointID(subscriptionID, resourceGroup, profileName, endpointName) + "/operationResults/delete?api-version=" + cdnAPIVersion,
		},
	}, nil
}

func (s *CDNService) handleOriginGroupRequest(ctx *service.RequestContext, route cdnRoute) (*service.Response, error) {
	if route.EndpointName == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.OriginGroupName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listOriginGroups(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateOriginGroup(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginGroupName, ctx.Body)
	case http.MethodGet:
		return s.getOriginGroup(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginGroupName)
	case http.MethodDelete:
		return s.deleteOriginGroup(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginGroupName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CDNService) createOrUpdateOriginGroup(subscriptionID, resourceGroup, profileName, endpointName, originGroupName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	endpoint, endpointExists := s.endpoints[parentKey]
	if !endpointExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	properties := cloneMap(input.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["resourceState"]; !ok {
		properties["resourceState"] = "Active"
	}
	if _, ok := properties["origins"]; !ok {
		properties["origins"] = []any{}
	}

	originGroup := OriginGroup{
		ID:         originGroupID(subscriptionID, resourceGroup, profileName, endpointName, originGroupName),
		Name:       originGroupName,
		Type:       "Microsoft.Cdn/profiles/endpoints/originGroups",
		Properties: properties,
	}

	key := originGroupKey(subscriptionID, resourceGroup, profileName, endpointName, originGroupName)
	_, existed := s.originGroups[key]
	s.originGroups[key] = originGroup
	endpoint.Properties["originGroups"] = s.originGroupsForEndpointLocked(parentKey)
	s.endpoints[parentKey] = endpoint

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, originGroup)
}

func (s *CDNService) getOriginGroup(subscriptionID, resourceGroup, profileName, endpointName, originGroupName string) (*service.Response, error) {
	s.mu.RLock()
	originGroup, ok := s.originGroups[originGroupKey(subscriptionID, resourceGroup, profileName, endpointName, originGroupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN origin group %q could not be found.", originGroupName))
	}
	return azurearm.JSONResponse(http.StatusOK, originGroup)
}

func (s *CDNService) listOriginGroups(subscriptionID, resourceGroup, profileName, endpointName string) (*service.Response, error) {
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)

	s.mu.RLock()
	_, parentExists := s.endpoints[parentKey]
	values := s.originGroupsForEndpointLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CDNService) deleteOriginGroup(subscriptionID, resourceGroup, profileName, endpointName, originGroupName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := originGroupKey(subscriptionID, resourceGroup, profileName, endpointName, originGroupName)
	if _, ok := s.originGroups[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.originGroups, key)
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if endpoint, ok := s.endpoints[parentKey]; ok {
		endpoint.Properties["originGroups"] = s.originGroupsForEndpointLocked(parentKey)
		s.endpoints[parentKey] = endpoint
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location": originGroupID(subscriptionID, resourceGroup, profileName, endpointName, originGroupName) + "/operationResults/delete?api-version=" + cdnAPIVersion,
		},
	}, nil
}

func (s *CDNService) handleOriginRequest(ctx *service.RequestContext, route cdnRoute) (*service.Response, error) {
	if route.EndpointName == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.OriginName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listOrigins(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateOrigin(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginName, ctx.Body)
	case http.MethodGet:
		return s.getOrigin(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginName)
	case http.MethodDelete:
		return s.deleteOrigin(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.OriginName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CDNService) createOrUpdateOrigin(subscriptionID, resourceGroup, profileName, endpointName, originName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	endpoint, endpointExists := s.endpoints[parentKey]
	if !endpointExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	properties := cloneMap(input.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["hostName"]; !ok {
		properties["hostName"] = originName
	}
	if _, ok := properties["httpPort"]; !ok {
		properties["httpPort"] = 80
	}
	if _, ok := properties["httpsPort"]; !ok {
		properties["httpsPort"] = 443
	}
	if _, ok := properties["priority"]; !ok {
		properties["priority"] = 1
	}
	if _, ok := properties["weight"]; !ok {
		properties["weight"] = 50
	}
	if _, ok := properties["enabled"]; !ok {
		properties["enabled"] = true
	}
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["resourceState"]; !ok {
		properties["resourceState"] = "Active"
	}

	origin := Origin{
		ID:         originID(subscriptionID, resourceGroup, profileName, endpointName, originName),
		Name:       originName,
		Type:       "Microsoft.Cdn/profiles/endpoints/origins",
		Properties: properties,
	}

	key := originKey(subscriptionID, resourceGroup, profileName, endpointName, originName)
	_, existed := s.origins[key]
	s.origins[key] = origin
	endpoint.Properties["origins"] = s.originsForEndpointLocked(parentKey)
	s.endpoints[parentKey] = endpoint

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, origin)
}

func (s *CDNService) getOrigin(subscriptionID, resourceGroup, profileName, endpointName, originName string) (*service.Response, error) {
	s.mu.RLock()
	origin, ok := s.origins[originKey(subscriptionID, resourceGroup, profileName, endpointName, originName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN origin %q could not be found.", originName))
	}
	return azurearm.JSONResponse(http.StatusOK, origin)
}

func (s *CDNService) listOrigins(subscriptionID, resourceGroup, profileName, endpointName string) (*service.Response, error) {
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)

	s.mu.RLock()
	_, parentExists := s.endpoints[parentKey]
	values := s.originsForEndpointLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CDNService) deleteOrigin(subscriptionID, resourceGroup, profileName, endpointName, originName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := originKey(subscriptionID, resourceGroup, profileName, endpointName, originName)
	if _, ok := s.origins[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.origins, key)
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if endpoint, ok := s.endpoints[parentKey]; ok {
		endpoint.Properties["origins"] = s.originsForEndpointLocked(parentKey)
		s.endpoints[parentKey] = endpoint
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location": originID(subscriptionID, resourceGroup, profileName, endpointName, originName) + "/operationResults/delete?api-version=" + cdnAPIVersion,
		},
	}, nil
}

func (s *CDNService) originsForEndpointLocked(parentKey string) []Origin {
	prefix := parentKey + "/"
	values := make([]Origin, 0)
	for key, origin := range s.origins {
		if strings.HasPrefix(key, prefix) {
			values = append(values, origin)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].ID < values[j].ID
		}
		return values[i].Name < values[j].Name
	})
	return values
}

func (s *CDNService) handleCustomDomainRequest(ctx *service.RequestContext, route cdnRoute) (*service.Response, error) {
	if route.EndpointName == "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
	}
	if route.CustomDomainAction != "" {
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		switch {
		case strings.EqualFold(route.CustomDomainAction, "enableCustomHttps"):
			return s.enableCustomDomainHTTPS(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.CustomDomainName, ctx.Body)
		case strings.EqualFold(route.CustomDomainAction, "disableCustomHttps"):
			return s.disableCustomDomainHTTPS(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.CustomDomainName)
		default:
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The CDN route is not implemented.")
		}
	}
	if route.CustomDomainName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listCustomDomains(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateCustomDomain(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.CustomDomainName, ctx.Body)
	case http.MethodGet:
		return s.getCustomDomain(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.CustomDomainName)
	case http.MethodDelete:
		return s.deleteCustomDomain(route.SubscriptionID, route.ResourceGroup, route.ProfileName, route.EndpointName, route.CustomDomainName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *CDNService) createOrUpdateCustomDomain(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	endpoint, endpointExists := s.endpoints[parentKey]
	if !endpointExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	properties := cloneMap(input.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	if _, ok := properties["hostName"]; !ok {
		properties["hostName"] = customDomainName
	}
	if _, ok := properties["provisioningState"]; !ok {
		properties["provisioningState"] = "Succeeded"
	}
	if _, ok := properties["resourceState"]; !ok {
		properties["resourceState"] = "Active"
	}
	if _, ok := properties["customHttpsProvisioningState"]; !ok {
		properties["customHttpsProvisioningState"] = "Enabling"
	}
	if _, ok := properties["customHttpsProvisioningSubstate"]; !ok {
		properties["customHttpsProvisioningSubstate"] = "PendingDomainControlValidationREquestApproval"
	}
	if _, ok := properties["validationData"]; !ok {
		properties["validationData"] = nil
	}

	customDomain := CustomDomain{
		ID:         customDomainID(subscriptionID, resourceGroup, profileName, endpointName, customDomainName),
		Name:       customDomainName,
		Type:       "Microsoft.Cdn/profiles/endpoints/customDomains",
		Properties: properties,
	}

	key := customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName)
	_, existed := s.customDomains[key]
	s.customDomains[key] = customDomain
	endpoint.Properties["customDomains"] = s.customDomainsForEndpointLocked(parentKey)
	s.endpoints[parentKey] = endpoint

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, customDomain)
}

func (s *CDNService) getCustomDomain(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string) (*service.Response, error) {
	s.mu.RLock()
	customDomain, ok := s.customDomains[customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN custom domain %q could not be found.", customDomainName))
	}
	return azurearm.JSONResponse(http.StatusOK, customDomain)
}

func (s *CDNService) listCustomDomains(subscriptionID, resourceGroup, profileName, endpointName string) (*service.Response, error) {
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)

	s.mu.RLock()
	_, parentExists := s.endpoints[parentKey]
	values := s.customDomainsForEndpointLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN endpoint %q could not be found.", endpointName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *CDNService) deleteCustomDomain(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName)
	if _, ok := s.customDomains[key]; !ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	delete(s.customDomains, key)
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if endpoint, ok := s.endpoints[parentKey]; ok {
		endpoint.Properties["customDomains"] = s.customDomainsForEndpointLocked(parentKey)
		s.endpoints[parentKey] = endpoint
	}
	return &service.Response{
		StatusCode: http.StatusAccepted,
		Headers: map[string]string{
			"Location": customDomainID(subscriptionID, resourceGroup, profileName, endpointName, customDomainName) + "/operationResults/delete?api-version=" + cdnAPIVersion,
		},
	}, nil
}

func (s *CDNService) customDomainsForEndpointLocked(parentKey string) []CustomDomain {
	prefix := parentKey + "/"
	values := make([]CustomDomain, 0)
	for key, customDomain := range s.customDomains {
		if strings.HasPrefix(key, prefix) {
			values = append(values, customDomain)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].ID < values[j].ID
		}
		return values[i].Name < values[j].Name
	})
	return values
}

func (s *CDNService) enableCustomDomainHTTPS(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string, body []byte) (*service.Response, error) {
	var httpsParameters map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &httpsParameters); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName)
	customDomain, ok := s.customDomains[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN custom domain %q could not be found.", customDomainName))
	}
	properties := cloneMap(customDomain.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	if httpsParameters != nil {
		properties["customHttpsParameters"] = httpsParameters
	}
	properties["customHttpsProvisioningState"] = "Enabled"
	properties["customHttpsProvisioningSubstate"] = "CertificateDeployed"
	properties["provisioningState"] = "Succeeded"
	properties["resourceState"] = "Active"

	customDomain.Properties = properties
	s.customDomains[key] = customDomain
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if endpoint, ok := s.endpoints[parentKey]; ok {
		endpoint.Properties["customDomains"] = s.customDomainsForEndpointLocked(parentKey)
		s.endpoints[parentKey] = endpoint
	}
	return azurearm.JSONResponse(http.StatusOK, customDomain)
}

func (s *CDNService) disableCustomDomainHTTPS(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName)
	customDomain, ok := s.customDomains[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("CDN custom domain %q could not be found.", customDomainName))
	}
	properties := cloneMap(customDomain.Properties)
	if properties == nil {
		properties = map[string]any{}
	}
	delete(properties, "customHttpsParameters")
	properties["customHttpsProvisioningState"] = "Disabled"
	properties["customHttpsProvisioningSubstate"] = "CertificateDeleted"
	properties["provisioningState"] = "Succeeded"
	properties["resourceState"] = "Active"

	customDomain.Properties = properties
	s.customDomains[key] = customDomain
	parentKey := endpointKey(subscriptionID, resourceGroup, profileName, endpointName)
	if endpoint, ok := s.endpoints[parentKey]; ok {
		endpoint.Properties["customDomains"] = s.customDomainsForEndpointLocked(parentKey)
		s.endpoints[parentKey] = endpoint
	}
	return azurearm.JSONResponse(http.StatusOK, customDomain)
}

func (s *CDNService) originGroupsForEndpointLocked(parentKey string) []OriginGroup {
	prefix := parentKey + "/"
	values := make([]OriginGroup, 0)
	for key, originGroup := range s.originGroups {
		if strings.HasPrefix(key, prefix) {
			values = append(values, originGroup)
		}
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].Name == values[j].Name {
			return values[i].ID < values[j].ID
		}
		return values[i].Name < values[j].Name
	})
	return values
}

type cdnRoute struct {
	SubscriptionID     string
	ResourceGroup      string
	ResourceType       string
	ProfileName        string
	ChildType          string
	EndpointName       string
	GrandchildType     string
	OriginGroupName    string
	OriginName         string
	CustomDomainName   string
	CustomDomainAction string
}

func parseRoute(escapedPath string) (cdnRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 5 || !strings.EqualFold(parts[0], "subscriptions") {
		return cdnRoute{}, false
	}
	if len(parts) >= 5 &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.Cdn") {
		route := cdnRoute{SubscriptionID: parts[1], ResourceType: parts[4]}
		switch len(parts) {
		case 5:
			return route, true
		default:
			return cdnRoute{}, false
		}
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Cdn") {
		return cdnRoute{}, false
	}
	route := cdnRoute{SubscriptionID: parts[1], ResourceGroup: parts[3], ResourceType: parts[6]}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.ProfileName = parts[7]
		return route, true
	case 9:
		route.ProfileName = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.ProfileName = parts[7]
		route.ChildType = parts[8]
		route.EndpointName = parts[9]
		return route, true
	case 11:
		route.ProfileName = parts[7]
		route.ChildType = parts[8]
		route.EndpointName = parts[9]
		route.GrandchildType = parts[10]
		return route, true
	case 12:
		route.ProfileName = parts[7]
		route.ChildType = parts[8]
		route.EndpointName = parts[9]
		route.GrandchildType = parts[10]
		if strings.EqualFold(route.GrandchildType, "originGroups") {
			route.OriginGroupName = parts[11]
		}
		if strings.EqualFold(route.GrandchildType, "origins") {
			route.OriginName = parts[11]
		}
		if strings.EqualFold(route.GrandchildType, "customDomains") {
			route.CustomDomainName = parts[11]
		}
		return route, true
	case 13:
		route.ProfileName = parts[7]
		route.ChildType = parts[8]
		route.EndpointName = parts[9]
		route.GrandchildType = parts[10]
		if strings.EqualFold(route.GrandchildType, "customDomains") {
			route.CustomDomainName = parts[11]
			route.CustomDomainAction = parts[12]
			return route, true
		}
		return cdnRoute{}, false
	default:
		return cdnRoute{}, false
	}
}

func profileID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cdn/profiles/" + name
}

func endpointID(subscriptionID, resourceGroup, profileName, endpointName string) string {
	return profileID(subscriptionID, resourceGroup, profileName) + "/endpoints/" + endpointName
}

func originGroupID(subscriptionID, resourceGroup, profileName, endpointName, originGroupName string) string {
	return endpointID(subscriptionID, resourceGroup, profileName, endpointName) + "/originGroups/" + originGroupName
}

func originID(subscriptionID, resourceGroup, profileName, endpointName, originName string) string {
	return endpointID(subscriptionID, resourceGroup, profileName, endpointName) + "/origins/" + originName
}

func customDomainID(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string) string {
	return endpointID(subscriptionID, resourceGroup, profileName, endpointName) + "/customDomains/" + customDomainName
}

func profileKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func endpointKey(subscriptionID, resourceGroup, profileName, endpointName string) string {
	return profileKey(subscriptionID, resourceGroup, profileName) + "/" + strings.ToLower(endpointName)
}

func originGroupKey(subscriptionID, resourceGroup, profileName, endpointName, originGroupName string) string {
	return endpointKey(subscriptionID, resourceGroup, profileName, endpointName) + "/" + strings.ToLower(originGroupName)
}

func originKey(subscriptionID, resourceGroup, profileName, endpointName, originName string) string {
	return endpointKey(subscriptionID, resourceGroup, profileName, endpointName) + "/" + strings.ToLower(originName)
}

func customDomainKey(subscriptionID, resourceGroup, profileName, endpointName, customDomainName string) string {
	return endpointKey(subscriptionID, resourceGroup, profileName, endpointName) + "/" + strings.ToLower(customDomainName)
}

func profileKeyPrefix(subscriptionID, resourceGroup string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"
}

func deterministicFrontDoorID(subscriptionID, resourceGroup, name string) string {
	sum := sha256.Sum256([]byte(profileKey(subscriptionID, resourceGroup, name)))
	hexValue := hex.EncodeToString(sum[:])[:32]
	return hexValue[:8] + "-" + hexValue[8:12] + "-" + hexValue[12:16] + "-" + hexValue[16:20] + "-" + hexValue[20:]
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
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func splitTripletName(name string) (string, string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 3 ||
		strings.TrimSpace(parts[0]) == "" ||
		strings.TrimSpace(parts[1]) == "" ||
		strings.TrimSpace(parts[2]) == "" {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
