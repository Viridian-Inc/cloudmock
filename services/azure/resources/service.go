package resources

import (
	"fmt"
	"net/http"
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

const apiVersion20210401 = "2021-04-01"

// ResourcesService implements the Azure Microsoft.Resources control-plane APIs.
type ResourcesService struct {
	mu            sync.RWMutex
	resourceGroup map[string]map[string]ResourceGroup
	resources     map[string]GenericResource
	deployments   map[string]map[string]Deployment
	deploymentOps map[string]int
	providers     map[string]ProviderManifest
	registrations map[string]map[string]string
	provisioners  []TemplateProvisioner
}

// New returns a Microsoft.Resources service backed by an in-memory store.
func New() *ResourcesService {
	return &ResourcesService{
		resourceGroup: make(map[string]map[string]ResourceGroup),
		resources:     make(map[string]GenericResource),
		deployments:   make(map[string]map[string]Deployment),
		deploymentOps: make(map[string]int),
		providers:     defaultProviders(),
		registrations: make(map[string]map[string]string),
	}
}

func (s *ResourcesService) Name() string { return "Microsoft.Resources" }

func (s *ResourcesService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.Resources/resourceGroups/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.Resources/resourceGroups/read"},
		{Name: "CheckExistence", Method: http.MethodHead, IAMAction: "azure:Microsoft.Resources/resourceGroups/read"},
		{Name: "List", Method: http.MethodGet, IAMAction: "azure:Microsoft.Resources/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Resources/resourceGroups/delete"},
		{Name: "CreateOrUpdateResource", Method: http.MethodPut, IAMAction: "azure:Microsoft.Resources/resources/write"},
		{Name: "GetResource", Method: http.MethodGet, IAMAction: "azure:Microsoft.Resources/resources/read"},
		{Name: "CheckResourceExistence", Method: http.MethodHead, IAMAction: "azure:Microsoft.Resources/resources/read"},
		{Name: "ListResources", Method: http.MethodGet, IAMAction: "azure:Microsoft.Resources/resources/read"},
		{Name: "DeleteResource", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Resources/resources/delete"},
		{Name: "CreateOrUpdateTags", Method: http.MethodPut, IAMAction: "azure:Microsoft.Resources/tags/write"},
		{Name: "UpdateTags", Method: http.MethodPatch, IAMAction: "azure:Microsoft.Resources/tags/write"},
		{Name: "GetTags", Method: http.MethodGet, IAMAction: "azure:Microsoft.Resources/tags/read"},
		{Name: "DeleteTags", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Resources/tags/delete"},
		{Name: "register", Method: http.MethodPost, IAMAction: "azure:Microsoft.Resources/providers/register/action"},
	}
}

func (s *ResourcesService) HealthCheck() error { return nil }

func (s *ResourcesService) SetTemplateProvisioner(provisioner TemplateProvisioner) {
	if provisioner == nil {
		return
	}
	s.provisioners = append(s.provisioners, provisioner)
}

// ServiceKeys returns the provider-aware registry keys handled by this service.
func (s *ResourcesService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/resourceGroups", APIVersion: apiVersion20210401},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/providers", APIVersion: apiVersion20210401},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/deployments", APIVersion: apiVersion20210401},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/resources", APIVersion: apiVersion20210401},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/tags", APIVersion: apiVersion20210401},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Resources/subscriptions", APIVersion: apiVersion20210401},
	}
}

func (s *ResourcesService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	if len(parts) < 2 || !strings.EqualFold(parts[0], "subscriptions") {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidResourceId", "Expected /subscriptions/{subscriptionId}.")
	}
	subscriptionID := parts[1]

	if pathHasSegment(parts, "providers") {
		return s.handleProviders(ctx, subscriptionID, parts)
	}
	if len(parts) == 3 && strings.EqualFold(parts[2], "resources") && ctx.RawRequest.Method == http.MethodGet {
		return s.listGenericResourcesBySubscription(ctx, subscriptionID)
	}
	if len(parts) >= 3 && strings.EqualFold(parts[2], "resourceGroups") {
		return s.handleResourceGroups(ctx, subscriptionID, parts)
	}

	return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", fmt.Sprintf("The path %q is not implemented.", ctx.RawRequest.URL.Path))
}

func pathHasSegment(parts []string, segment string) bool {
	return segmentIndex(parts, segment) >= 0
}

func segmentIndex(parts []string, segment string) int {
	for i, part := range parts {
		if strings.EqualFold(part, segment) {
			return i
		}
	}
	return -1
}

func lastSegmentIndex(parts []string, segment string) int {
	for i := len(parts) - 1; i >= 0; i-- {
		if strings.EqualFold(parts[i], segment) {
			return i
		}
	}
	return -1
}

func (s *ResourcesService) handleResourceGroups(ctx *service.RequestContext, subscriptionID string, parts []string) (*service.Response, error) {
	switch {
	case len(parts) == 3 && ctx.RawRequest.Method == http.MethodGet:
		return s.listResourceGroups(ctx, subscriptionID)
	case len(parts) == 5 && strings.EqualFold(parts[4], "resources") && ctx.RawRequest.Method == http.MethodGet:
		return s.listGenericResources(ctx, subscriptionID, parts[3])
	case len(parts) == 4:
		name := parts[3]
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createOrUpdateResourceGroup(subscriptionID, name, ctx.Body)
		case http.MethodGet:
			if ctx.RawRequest.URL.Query().Get("$operationStatus") == "delete-resource-group" {
				return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
			}
			return s.getResourceGroup(subscriptionID, name)
		case http.MethodHead:
			return s.checkResourceGroupExists(subscriptionID, name)
		case http.MethodDelete:
			return s.deleteResourceGroup(ctx, subscriptionID, name)
		}
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The resource group route is not implemented.")
}

func (s *ResourcesService) createOrUpdateResourceGroup(subscriptionID, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location  string         `json:"location"`
		ManagedBy string         `json:"managedBy"`
		Tags      map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.Location == "" {
		input.Location = "eastus"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.resourceGroup[subscriptionID] == nil {
		s.resourceGroup[subscriptionID] = make(map[string]ResourceGroup)
	}

	key := strings.ToLower(name)
	_, existed := s.resourceGroup[subscriptionID][key]
	rg := ResourceGroup{
		ID:        "/subscriptions/" + subscriptionID + "/resourceGroups/" + name,
		Name:      name,
		Type:      "Microsoft.Resources/resourceGroups",
		Location:  input.Location,
		ManagedBy: input.ManagedBy,
		Tags:      stringifyTags(input.Tags),
		Properties: ResourceGroupProperties{
			ProvisioningState: "Succeeded",
		},
	}
	s.resourceGroup[subscriptionID][key] = rg

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, rg)
}

func (s *ResourcesService) getResourceGroup(subscriptionID, name string) (*service.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rg, ok := s.resourceGroup[subscriptionID][strings.ToLower(name)]; ok {
		return azurearm.JSONResponse(http.StatusOK, rg)
	}
	return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", name))
}

func (s *ResourcesService) checkResourceGroupExists(subscriptionID, name string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.resourceGroup[subscriptionID][strings.ToLower(name)]
	s.mu.RUnlock()
	if ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	return &service.Response{StatusCode: http.StatusNotFound}, nil
}

func (s *ResourcesService) listResourceGroups(ctx *service.RequestContext, subscriptionID string) (*service.Response, error) {
	options := genericResourceListOptionsFromRequest(ctx.RawRequest)

	s.mu.RLock()
	values := make([]ResourceGroup, 0, len(s.resourceGroup[subscriptionID]))
	for _, rg := range s.resourceGroup[subscriptionID] {
		if resourceGroupMatchesFilter(rg, options.Filter) {
			values = append(values, rg)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })

	start := options.Skip
	if start > len(values) {
		start = len(values)
	}
	end := len(values)
	if options.Top > 0 && start+options.Top < end {
		end = start + options.Top
	}
	body := map[string]any{"value": values[start:end]}
	if options.Top > 0 && end < len(values) {
		body["nextLink"] = armListNextLink(ctx.RawRequest, end)
	}
	return azurearm.JSONResponse(http.StatusOK, body)
}

func resourceGroupMatchesFilter(rg ResourceGroup, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	var tagName, tagValue string
	var hasTagName, hasTagValue bool
	for _, clause := range splitARMFilterClauses(filter) {
		field, value, ok := parseARMEqualsClause(clause)
		if !ok {
			continue
		}
		switch strings.ToLower(field) {
		case "tagname":
			tagName, hasTagName = value, true
		case "tagvalue":
			tagValue, hasTagValue = value, true
		}
	}
	if hasTagName && hasTagValue {
		return rg.Tags[tagName] == tagValue
	}
	if hasTagName {
		_, exists := rg.Tags[tagName]
		return exists
	}
	if hasTagValue {
		for _, value := range rg.Tags {
			if value == tagValue {
				return true
			}
		}
		return false
	}
	return true
}

func (s *ResourcesService) deleteResourceGroup(ctx *service.RequestContext, subscriptionID, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.resourceGroup[subscriptionID]
	key := strings.ToLower(name)
	if groups == nil {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", name))
	}
	if _, ok := groups[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", name))
	}
	delete(groups, key)
	s.deleteResourceGroupChildrenLocked(subscriptionID, name)

	operation := *ctx.RawRequest.URL
	query := operation.Query()
	query.Set("$operationStatus", "delete-resource-group")
	operation.RawQuery = query.Encode()

	resp, err := azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{
		"Azure-AsyncOperation": operation.String(),
		"Retry-After":          "1",
	}
	return resp, nil
}

func (s *ResourcesService) deleteResourceGroupChildrenLocked(subscriptionID, resourceGroup string) {
	prefix := strings.ToLower("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/")
	for resourceID := range s.resources {
		if strings.HasPrefix(resourceID, prefix) {
			delete(s.resources, resourceID)
		}
	}
	delete(s.deployments, deploymentScopeKey(subscriptionID, resourceGroup))
}

func (s *ResourcesService) handleProviders(ctx *service.RequestContext, subscriptionID string, parts []string) (*service.Response, error) {
	providerIndex := segmentIndex(parts, "providers")
	if providerIndex < 0 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The provider route is not implemented.")
	}
	if len(parts) == providerIndex+1 && ctx.RawRequest.Method == http.MethodGet {
		return s.listProviders(subscriptionID)
	}
	if len(parts) < providerIndex+2 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The provider route is not implemented.")
	}

	if tagsProviderIndex := lastSegmentIndex(parts, "providers"); tagsProviderIndex >= 0 &&
		strings.EqualFold(parts[tagsProviderIndex+1], "Microsoft.Resources") &&
		tagsProviderIndex+2 < len(parts) &&
		strings.EqualFold(parts[tagsProviderIndex+2], "tags") {
		return s.handleTags(ctx, subscriptionID, parts, tagsProviderIndex)
	}

	namespace := parts[providerIndex+1]
	if strings.EqualFold(namespace, "Microsoft.Resources") && len(parts) >= providerIndex+3 && strings.EqualFold(parts[providerIndex+2], "deployments") {
		return s.handleDeployments(ctx, subscriptionID, parts, providerIndex)
	}
	if len(parts) == providerIndex+2 && ctx.RawRequest.Method == http.MethodGet {
		return s.getProvider(subscriptionID, namespace)
	}
	if len(parts) == providerIndex+3 && strings.EqualFold(parts[providerIndex+2], "register") && ctx.RawRequest.Method == http.MethodPost {
		return s.registerProvider(subscriptionID, namespace)
	}
	if genericResourceRoute(parts, providerIndex) {
		return s.handleGenericResource(ctx, subscriptionID, parts, providerIndex)
	}

	return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The provider route is not implemented.")
}

func genericResourceRoute(parts []string, providerIndex int) bool {
	return providerIndex >= 4 &&
		len(parts) >= providerIndex+4 &&
		len(parts[providerIndex+2:])%2 == 0 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "resourceGroups")
}

func (s *ResourcesService) handleGenericResource(ctx *service.RequestContext, subscriptionID string, parts []string, providerIndex int) (*service.Response, error) {
	identity, ok := genericResourceIdentity(subscriptionID, parts, providerIndex)
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The resource route is not implemented.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateGenericResource(identity, ctx.Body)
	case http.MethodGet:
		if ctx.RawRequest.URL.Query().Get("$operationStatus") == "delete-resource" {
			return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
		}
		return s.getGenericResource(identity.ID)
	case http.MethodHead:
		return s.checkGenericResourceExists(identity.ID)
	case http.MethodDelete:
		return s.deleteGenericResource(ctx, identity.ID)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

type genericResourceRouteIdentity struct {
	SubscriptionID string
	ResourceGroup  string
	ID             string
	Name           string
	Type           string
}

func genericResourceIdentity(subscriptionID string, parts []string, providerIndex int) (genericResourceRouteIdentity, bool) {
	if providerIndex < 4 || providerIndex+3 >= len(parts) {
		return genericResourceRouteIdentity{}, false
	}
	if !strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		parts[1] != subscriptionID {
		return genericResourceRouteIdentity{}, false
	}
	tail := parts[providerIndex+2:]
	if len(tail) == 0 || len(tail)%2 != 0 {
		return genericResourceRouteIdentity{}, false
	}

	resourceTypes := make([]string, 0, len(tail)/2)
	resourceNames := make([]string, 0, len(tail)/2)
	var id strings.Builder
	id.WriteString("/subscriptions/")
	id.WriteString(subscriptionID)
	id.WriteString("/resourceGroups/")
	id.WriteString(parts[3])
	id.WriteString("/providers/")
	id.WriteString(parts[providerIndex+1])
	for i := 0; i < len(tail); i += 2 {
		resourceTypes = append(resourceTypes, tail[i])
		resourceNames = append(resourceNames, tail[i+1])
		id.WriteString("/")
		id.WriteString(tail[i])
		id.WriteString("/")
		id.WriteString(tail[i+1])
	}

	return genericResourceRouteIdentity{
		SubscriptionID: subscriptionID,
		ResourceGroup:  parts[3],
		ID:             id.String(),
		Name:           strings.Join(resourceNames, "/"),
		Type:           parts[providerIndex+1] + "/" + strings.Join(resourceTypes, "/"),
	}, true
}

func (s *ResourcesService) createOrUpdateGenericResource(identity genericResourceRouteIdentity, body []byte) (*service.Response, error) {
	var input map[string]any
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input == nil {
		input = map[string]any{}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.resourceGroup[identity.SubscriptionID]
	rg, ok := groups[strings.ToLower(identity.ResourceGroup)]
	if groups == nil || !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", identity.ResourceGroup))
	}

	location := stringValue(input["location"])
	if location == "" {
		location = rg.Location
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	createdTime := now
	key := genericResourceKey(identity.ID)
	if existing, existed := s.resources[key]; existed && existing.createdTime != "" {
		createdTime = existing.createdTime
	}

	resource := GenericResource{
		ID:               identity.ID,
		Name:             identity.Name,
		Type:             identity.Type,
		Location:         location,
		Tags:             stringifyTags(asMap(input["tags"])),
		Kind:             stringValue(input["kind"]),
		ManagedBy:        stringValue(input["managedBy"]),
		ExtendedLocation: asMap(input["extendedLocation"]),
		Identity:         asMap(input["identity"]),
		Plan:             asMap(input["plan"]),
		SKU:              asMap(input["sku"]),
		Properties:       asMap(input["properties"]),
		createdTime:      createdTime,
		changedTime:      now,
	}

	_, existed := s.resources[key]
	s.resources[key] = resource

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, resource)
}

func (s *ResourcesService) getGenericResource(resourceID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.resources[genericResourceKey(resourceID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Resource %q could not be found.", resourceID))
	}
	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *ResourcesService) checkGenericResourceExists(resourceID string) (*service.Response, error) {
	s.mu.RLock()
	_, ok := s.resources[genericResourceKey(resourceID)]
	s.mu.RUnlock()
	if ok {
		return &service.Response{StatusCode: http.StatusNoContent}, nil
	}
	return &service.Response{StatusCode: http.StatusNotFound}, nil
}

func (s *ResourcesService) listGenericResources(ctx *service.RequestContext, subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower("/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/")
	options := genericResourceListOptionsFromRequest(ctx.RawRequest)

	s.mu.RLock()
	values := make([]GenericResource, 0)
	for key, resource := range s.resources {
		if strings.HasPrefix(key, prefix) && genericResourceMatchesFilter(resource, resourceGroup, options.Filter) {
			values = append(values, applyGenericResourceListOptions(resource, options))
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool {
		if values[i].Type == values[j].Type {
			return values[i].Name < values[j].Name
		}
		return values[i].Type < values[j].Type
	})

	start := options.Skip
	if start > len(values) {
		start = len(values)
	}
	end := len(values)
	if options.Top > 0 && start+options.Top < end {
		end = start + options.Top
	}
	body := map[string]any{"value": values[start:end]}
	if options.Top > 0 && end < len(values) {
		body["nextLink"] = armListNextLink(ctx.RawRequest, end)
	}
	return azurearm.JSONResponse(http.StatusOK, body)
}

func (s *ResourcesService) listGenericResourcesBySubscription(ctx *service.RequestContext, subscriptionID string) (*service.Response, error) {
	prefix := strings.ToLower("/subscriptions/" + subscriptionID + "/resourceGroups/")
	options := genericResourceListOptionsFromRequest(ctx.RawRequest)

	s.mu.RLock()
	values := make([]GenericResource, 0)
	for key, resource := range s.resources {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		resourceGroup := genericResourceGroupFromID(resource.ID)
		if genericResourceMatchesFilter(resource, resourceGroup, options.Filter) {
			values = append(values, applyGenericResourceListOptions(resource, options))
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool {
		if values[i].Type == values[j].Type {
			return values[i].ID < values[j].ID
		}
		return values[i].Type < values[j].Type
	})

	start := options.Skip
	if start > len(values) {
		start = len(values)
	}
	end := len(values)
	if options.Top > 0 && start+options.Top < end {
		end = start + options.Top
	}
	body := map[string]any{"value": values[start:end]}
	if options.Top > 0 && end < len(values) {
		body["nextLink"] = armListNextLink(ctx.RawRequest, end)
	}
	return azurearm.JSONResponse(http.StatusOK, body)
}

type genericResourceListOptions struct {
	Filter                  string
	OmitTags                bool
	Top                     int
	Skip                    int
	ExpandCreatedTime       bool
	ExpandChangedTime       bool
	ExpandProvisioningState bool
}

func genericResourceListOptionsFromRequest(req *http.Request) genericResourceListOptions {
	query := req.URL.Query()
	options := genericResourceListOptions{
		Filter: strings.TrimSpace(query.Get("$filter")),
	}
	options.OmitTags = strings.Contains(strings.ToLower(options.Filter), "tagname") && strings.Contains(strings.ToLower(options.Filter), "tagvalue")
	if top, err := strconv.Atoi(query.Get("$top")); err == nil && top > 0 {
		options.Top = top
	}
	if skip, err := strconv.Atoi(query.Get("$skiptoken")); err == nil && skip > 0 {
		options.Skip = skip
	}
	for _, field := range strings.Split(query.Get("$expand"), ",") {
		switch strings.ToLower(strings.TrimSpace(field)) {
		case "createdtime":
			options.ExpandCreatedTime = true
		case "changedtime":
			options.ExpandChangedTime = true
		case "provisioningstate":
			options.ExpandProvisioningState = true
		}
	}
	return options
}

func applyGenericResourceListOptions(resource GenericResource, options genericResourceListOptions) GenericResource {
	if options.OmitTags {
		resource.Tags = nil
	}
	if options.ExpandCreatedTime {
		resource.CreatedTime = resource.createdTime
	}
	if options.ExpandChangedTime {
		resource.ChangedTime = resource.changedTime
	}
	if options.ExpandProvisioningState {
		resource.ProvisioningState = genericResourceProvisioningState(resource)
	}
	return resource
}

func genericResourceProvisioningState(resource GenericResource) string {
	if state := stringValue(resource.Properties["provisioningState"]); state != "" {
		return state
	}
	return "Succeeded"
}

func genericResourceMatchesFilter(resource GenericResource, resourceGroup, filter string) bool {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return true
	}
	var tagName, tagValue string
	var hasTagName, hasTagValue bool
	for _, andGroup := range splitARMFilterClauses(filter) {
		groupHasNonTagClause := false
		groupMatched := false
		for _, clause := range splitARMFilterExpression(andGroup, " or ") {
			if field, op, value, ok := parseARMComparisonClause(clause); ok && op == "eq" {
				switch strings.ToLower(field) {
				case "tagname":
					tagName, hasTagName = value, true
					continue
				case "tagvalue":
					tagValue, hasTagValue = value, true
					continue
				}
			}
			groupHasNonTagClause = true
			if genericResourceMatchesFilterClause(resource, resourceGroup, clause) {
				groupMatched = true
			}
		}
		if groupHasNonTagClause && !groupMatched {
			return false
		}
	}
	if hasTagName && hasTagValue {
		return resource.Tags[tagName] == tagValue
	}
	if hasTagName {
		_, exists := resource.Tags[tagName]
		return exists
	}
	if hasTagValue {
		for _, value := range resource.Tags {
			if value == tagValue {
				return true
			}
		}
		return false
	}
	return true
}

func splitARMFilterClauses(filter string) []string {
	return splitARMFilterExpression(filter, " and ")
}

func splitARMFilterExpression(filter, operator string) []string {
	parts := strings.Split(filter, operator)
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

func genericResourceMatchesFilterClause(resource GenericResource, resourceGroup, clause string) bool {
	lower := strings.ToLower(clause)
	if strings.HasPrefix(lower, "startswith(") && strings.HasSuffix(clause, ")") {
		args := strings.TrimSpace(clause[len("startswith(") : len(clause)-1])
		property, prefix, ok := parseStartsWithArgs(args)
		if !ok {
			return true
		}
		switch strings.ToLower(property) {
		case "tagname":
			for key := range resource.Tags {
				if strings.HasPrefix(key, prefix) {
					return true
				}
			}
			return false
		default:
			return true
		}
	}
	if strings.HasPrefix(lower, "substringof(") && strings.HasSuffix(clause, ")") {
		args := strings.TrimSpace(clause[len("substringof(") : len(clause)-1])
		value, property, ok := parseSubstringOfArgs(args)
		if !ok {
			return true
		}
		switch strings.ToLower(property) {
		case "name":
			return strings.Contains(strings.ToLower(resource.Name), strings.ToLower(value))
		case "resourcegroup":
			return strings.Contains(strings.ToLower(resourceGroup), strings.ToLower(value))
		default:
			return true
		}
	}

	field, op, value, ok := parseARMComparisonClause(clause)
	if !ok {
		return true
	}
	matches := false
	switch strings.ToLower(field) {
	case "location":
		matches = strings.EqualFold(resource.Location, value)
	case "resourcetype":
		matches = strings.EqualFold(resource.Type, value)
	case "name":
		matches = strings.EqualFold(resource.Name, value)
	case "resourcegroup":
		matches = strings.EqualFold(resourceGroup, value)
	case "managedby":
		matches = strings.EqualFold(resource.ManagedBy, value)
	case "identity":
		matches = strings.EqualFold(stringValue(resource.Identity["type"]), value)
	case "identity/principalid":
		matches = strings.EqualFold(stringValue(resource.Identity["principalId"]), value)
	case "plan":
		matches = genericResourcePlanMatches(resource.Plan, value)
	case "plan/publisher":
		matches = strings.EqualFold(stringValue(resource.Plan["publisher"]), value)
	case "plan/product":
		matches = strings.EqualFold(stringValue(resource.Plan["product"]), value)
	case "plan/name":
		matches = strings.EqualFold(stringValue(resource.Plan["name"]), value)
	case "plan/version":
		matches = strings.EqualFold(stringValue(resource.Plan["version"]), value)
	case "plan/promotioncode":
		matches = strings.EqualFold(stringValue(resource.Plan["promotionCode"]), value)
	case "tagname":
		_, exists := resource.Tags[value]
		matches = exists
	case "tagvalue":
		for _, tagValue := range resource.Tags {
			if tagValue == value {
				matches = true
				break
			}
		}
	default:
		return true
	}
	if op == "ne" {
		return !matches
	}
	return matches
}

func genericResourcePlanMatches(plan map[string]any, value string) bool {
	for _, field := range []string{"name", "publisher", "product", "version", "promotionCode"} {
		if strings.EqualFold(stringValue(plan[field]), value) {
			return true
		}
	}
	return false
}

func genericResourceGroupFromID(resourceID string) string {
	parts := splitPath(resourceID)
	for i := 0; i+1 < len(parts); i++ {
		if strings.EqualFold(parts[i], "resourceGroups") {
			return parts[i+1]
		}
	}
	return ""
}

func parseARMEqualsClause(clause string) (string, string, bool) {
	field, op, value, ok := parseARMComparisonClause(clause)
	return field, value, ok && op == "eq"
}

func parseARMComparisonClause(clause string) (string, string, string, bool) {
	op := "eq"
	parts := strings.SplitN(clause, " eq ", 2)
	if len(parts) != 2 {
		op = "ne"
		parts = strings.SplitN(clause, " ne ", 2)
	}
	if len(parts) != 2 {
		return "", "", "", false
	}
	return strings.TrimSpace(parts[0]), op, trimARMQuotedString(parts[1]), true
}

func parseSubstringOfArgs(args string) (string, string, bool) {
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return trimARMQuotedString(parts[0]), strings.TrimSpace(parts[1]), true
}

func parseStartsWithArgs(args string) (string, string, bool) {
	parts := strings.SplitN(args, ",", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), trimARMQuotedString(parts[1]), true
}

func trimARMQuotedString(value string) string {
	value = strings.TrimSpace(value)
	if len(value) >= 2 && value[0] == '\'' && value[len(value)-1] == '\'' {
		return value[1 : len(value)-1]
	}
	return value
}

func armListNextLink(req *http.Request, skip int) string {
	next := *req.URL
	query := next.Query()
	query.Set("$skiptoken", strconv.Itoa(skip))
	next.RawQuery = query.Encode()
	return next.String()
}

func (s *ResourcesService) deleteGenericResource(ctx *service.RequestContext, resourceID string) (*service.Response, error) {
	s.mu.Lock()
	delete(s.resources, genericResourceKey(resourceID))
	s.mu.Unlock()

	operation := *ctx.RawRequest.URL
	query := operation.Query()
	query.Set("$operationStatus", "delete-resource")
	operation.RawQuery = query.Encode()

	resp, err := azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
	if err != nil {
		return nil, err
	}
	resp.Headers = map[string]string{
		"Azure-AsyncOperation": operation.String(),
		"Retry-After":          "1",
	}
	return resp, nil
}

func genericResourceKey(resourceID string) string {
	return strings.ToLower(resourceID)
}

func (s *ResourcesService) handleTags(ctx *service.RequestContext, subscriptionID string, parts []string, providerIndex int) (*service.Response, error) {
	if len(parts) != providerIndex+4 || !strings.EqualFold(parts[providerIndex+3], "default") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The tags route is not implemented.")
	}
	scopeParts := parts[:providerIndex]
	if len(scopeParts) != 4 ||
		!strings.EqualFold(scopeParts[0], "subscriptions") ||
		!strings.EqualFold(scopeParts[2], "resourceGroups") {
		if scopedProviderIndex := segmentIndex(scopeParts, "providers"); scopedProviderIndex >= 0 {
			identity, ok := genericResourceIdentity(subscriptionID, scopeParts, scopedProviderIndex)
			if !ok {
				return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The tags scope is not implemented.")
			}
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.createOrUpdateTagsAtGenericResourceScope(identity.ID, ctx.Body)
			case http.MethodPatch:
				return s.updateTagsAtGenericResourceScope(identity.ID, ctx.Body)
			case http.MethodGet:
				return s.getTagsAtGenericResourceScope(identity.ID)
			case http.MethodDelete:
				return s.deleteTagsAtGenericResourceScope(identity.ID)
			default:
				return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
			}
		}
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "Only resource group and generic resource tag scopes are implemented.")
	}

	resourceGroupName := scopeParts[3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateTagsAtResourceGroupScope(subscriptionID, resourceGroupName, ctx.Body)
	case http.MethodPatch:
		return s.updateTagsAtResourceGroupScope(subscriptionID, resourceGroupName, ctx.Body)
	case http.MethodGet:
		return s.getTagsAtResourceGroupScope(subscriptionID, resourceGroupName)
	case http.MethodDelete:
		return s.deleteTagsAtResourceGroupScope(subscriptionID, resourceGroupName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *ResourcesService) createOrUpdateTagsAtResourceGroupScope(subscriptionID, resourceGroupName string, body []byte) (*service.Response, error) {
	var input struct {
		Properties TagsProperties `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.resourceGroup[subscriptionID]
	key := strings.ToLower(resourceGroupName)
	if groups == nil {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg, ok := groups[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg.Tags = cloneTags(input.Properties.Tags)
	groups[key] = rg

	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(rg.ID, rg.Tags))
}

func (s *ResourcesService) updateTagsAtResourceGroupScope(subscriptionID, resourceGroupName string, body []byte) (*service.Response, error) {
	input, errResp, err := parseTagsPatchRequest(body)
	if errResp != nil || err != nil {
		return errResp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.resourceGroup[subscriptionID]
	key := strings.ToLower(resourceGroupName)
	if groups == nil {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg, ok := groups[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	tags, errResp, err := applyTagsPatch(rg.Tags, input.Operation, input.Properties.Tags)
	if errResp != nil || err != nil {
		return errResp, err
	}
	rg.Tags = tags
	groups[key] = rg

	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(rg.ID, rg.Tags))
}

func (s *ResourcesService) getTagsAtResourceGroupScope(subscriptionID, resourceGroupName string) (*service.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	groups := s.resourceGroup[subscriptionID]
	key := strings.ToLower(resourceGroupName)
	if groups == nil {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg, ok := groups[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(rg.ID, rg.Tags))
}

func (s *ResourcesService) deleteTagsAtResourceGroupScope(subscriptionID, resourceGroupName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	groups := s.resourceGroup[subscriptionID]
	key := strings.ToLower(resourceGroupName)
	if groups == nil {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg, ok := groups[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceGroupNotFound", fmt.Sprintf("Resource group %q could not be found.", resourceGroupName))
	}
	rg.Tags = nil
	groups[key] = rg

	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ResourcesService) createOrUpdateTagsAtGenericResourceScope(resourceID string, body []byte) (*service.Response, error) {
	var input struct {
		Properties TagsProperties `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resource, ok := s.resources[genericResourceKey(resourceID)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Resource %q could not be found.", resourceID))
	}
	resource.Tags = cloneTags(input.Properties.Tags)
	s.resources[genericResourceKey(resourceID)] = resource

	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(resource.ID, resource.Tags))
}

func (s *ResourcesService) updateTagsAtGenericResourceScope(resourceID string, body []byte) (*service.Response, error) {
	input, errResp, err := parseTagsPatchRequest(body)
	if errResp != nil || err != nil {
		return errResp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	resource, ok := s.resources[genericResourceKey(resourceID)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Resource %q could not be found.", resourceID))
	}
	tags, errResp, err := applyTagsPatch(resource.Tags, input.Operation, input.Properties.Tags)
	if errResp != nil || err != nil {
		return errResp, err
	}
	resource.Tags = tags
	s.resources[genericResourceKey(resourceID)] = resource

	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(resource.ID, resource.Tags))
}

func (s *ResourcesService) getTagsAtGenericResourceScope(resourceID string) (*service.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resource, ok := s.resources[genericResourceKey(resourceID)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Resource %q could not be found.", resourceID))
	}
	return azurearm.JSONResponse(http.StatusOK, tagsResourceForScope(resource.ID, resource.Tags))
}

func (s *ResourcesService) deleteTagsAtGenericResourceScope(resourceID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	resource, ok := s.resources[genericResourceKey(resourceID)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Resource %q could not be found.", resourceID))
	}
	resource.Tags = nil
	s.resources[genericResourceKey(resourceID)] = resource

	return &service.Response{StatusCode: http.StatusOK}, nil
}

func (s *ResourcesService) listProviders(subscriptionID string) (*service.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	values := make([]ProviderManifest, 0, len(s.providers))
	for _, provider := range s.providers {
		values = append(values, s.providerManifestForSubscriptionLocked(subscriptionID, provider))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Namespace < values[j].Namespace })

	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *ResourcesService) getProvider(subscriptionID, namespace string) (*service.Response, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	provider, ok := s.providers[strings.ToLower(namespace)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ProviderNotFound", fmt.Sprintf("The resource provider %q could not be found.", namespace))
	}
	return azurearm.JSONResponse(http.StatusOK, s.providerManifestForSubscriptionLocked(subscriptionID, provider))
}

func (s *ResourcesService) registerProvider(subscriptionID, namespace string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	provider, ok := s.providers[strings.ToLower(namespace)]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ProviderNotFound", fmt.Sprintf("The resource provider %q could not be found.", namespace))
	}
	if s.registrations[subscriptionID] == nil {
		s.registrations[subscriptionID] = make(map[string]string)
	}
	s.registrations[subscriptionID][strings.ToLower(provider.Namespace)] = "Registered"
	return azurearm.JSONResponse(http.StatusOK, s.providerManifestForSubscriptionLocked(subscriptionID, provider))
}

func (s *ResourcesService) providerManifestForSubscriptionLocked(subscriptionID string, provider ProviderManifest) ProviderManifest {
	provider.ID = "/subscriptions/" + subscriptionID + "/providers/" + provider.Namespace
	if provider.RegistrationPolicy == "" {
		provider.RegistrationPolicy = "RegistrationRequired"
		if strings.EqualFold(provider.Namespace, "Microsoft.Resources") {
			provider.RegistrationPolicy = "RegistrationFree"
		}
	}
	provider.RegistrationState = s.registrationStateLocked(subscriptionID, provider.Namespace)
	return provider
}

func (s *ResourcesService) registrationStateLocked(subscriptionID, namespace string) string {
	if strings.EqualFold(namespace, "Microsoft.Resources") {
		return "Registered"
	}
	if byProvider := s.registrations[subscriptionID]; byProvider != nil {
		if state := byProvider[strings.ToLower(namespace)]; state != "" {
			return state
		}
	}
	return "NotRegistered"
}

func splitPath(path string) []string {
	path = strings.Trim(path, "/")
	if path == "" {
		return nil
	}
	raw := strings.Split(path, "/")
	parts := raw[:0]
	for _, part := range raw {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func stringifyTags(tags map[string]any) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for k, v := range tags {
		out[k] = fmt.Sprint(v)
	}
	return out
}

func cloneTags(tags map[string]string) map[string]string {
	if len(tags) == 0 {
		return nil
	}
	out := make(map[string]string, len(tags))
	for key, value := range tags {
		out[key] = value
	}
	return out
}

type tagsPatchRequest struct {
	Operation  string         `json:"operation"`
	Properties TagsProperties `json:"properties"`
}

func parseTagsPatchRequest(body []byte) (tagsPatchRequest, *service.Response, error) {
	var input tagsPatchRequest
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			resp, respErr := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
			return input, resp, respErr
		}
	}
	if input.Operation == "" {
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidTagsPatchOperation", "The tags patch operation is required.")
		return input, resp, err
	}
	return input, nil, nil
}

func applyTagsPatch(existing map[string]string, operation string, tags map[string]string) (map[string]string, *service.Response, error) {
	switch strings.ToLower(operation) {
	case "replace":
		return cloneTags(tags), nil, nil
	case "merge":
		merged := cloneTags(existing)
		if merged == nil {
			merged = map[string]string{}
		}
		for key, value := range tags {
			merged[key] = value
		}
		if len(merged) == 0 {
			return nil, nil, nil
		}
		return merged, nil, nil
	case "delete":
		updated := cloneTags(existing)
		for key, value := range tags {
			if value == "" || updated[key] == value {
				delete(updated, key)
			}
		}
		if len(updated) == 0 {
			return nil, nil, nil
		}
		return updated, nil, nil
	default:
		resp, err := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidTagsPatchOperation", fmt.Sprintf("The tags patch operation %q is not supported.", operation))
		return nil, resp, err
	}
}

func tagsResourceForScope(scope string, tags map[string]string) TagsResource {
	return TagsResource{
		ID:   scope + "/providers/Microsoft.Resources/tags/default",
		Name: "default",
		Type: "Microsoft.Resources/tags",
		Properties: TagsProperties{
			Tags: cloneTags(tags),
		},
	}
}
