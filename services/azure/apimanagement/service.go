package apimanagement

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/azurearm"
	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

var apiManagementAPIVersions = []string{"2024-05-01"}

// APIManagementService implements the APIM ARM service-resource lifecycle.
type APIManagementService struct {
	mu            sync.RWMutex
	services      map[string]ServiceResource
	apis          map[string]map[string]any
	operations    map[string]map[string]any
	policies      map[string]map[string]any
	products      map[string]map[string]any
	productAPIs   map[string]string
	subscriptions map[string]map[string]any
	namedValues   map[string]map[string]any
	backends      map[string]map[string]any
	httpClient    *http.Client
}

func New() *APIManagementService {
	return &APIManagementService{
		services:      make(map[string]ServiceResource),
		apis:          make(map[string]map[string]any),
		operations:    make(map[string]map[string]any),
		policies:      make(map[string]map[string]any),
		products:      make(map[string]map[string]any),
		productAPIs:   make(map[string]string),
		subscriptions: make(map[string]map[string]any),
		namedValues:   make(map[string]map[string]any),
		backends:      make(map[string]map[string]any),
		httpClient:    http.DefaultClient,
	}
}

func (s *APIManagementService) Name() string { return "Microsoft.ApiManagement" }

func (s *APIManagementService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateService", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/write"},
		{Name: "GetService", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/read"},
		{Name: "ListServices", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/read"},
		{Name: "DeleteService", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/delete"},
		{Name: "CreateOrUpdateAPI", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/apis/write"},
		{Name: "GetAPI", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/apis/read"},
		{Name: "ListAPIs", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/apis/read"},
		{Name: "DeleteAPI", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/apis/delete"},
		{Name: "CreateOrUpdateOperation", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/apis/operations/write"},
		{Name: "GetOperation", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/apis/operations/read"},
		{Name: "ListOperations", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/apis/operations/read"},
		{Name: "DeleteOperation", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/apis/operations/delete"},
		{Name: "CreateOrUpdatePolicy", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/policies/write"},
		{Name: "GetPolicy", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/policies/read"},
		{Name: "ListPolicies", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/policies/read"},
		{Name: "DeletePolicy", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/policies/delete"},
		{Name: "GatewayProxy", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/gateway/action"},
		{Name: "CreateOrUpdateProduct", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/products/write"},
		{Name: "GetProduct", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/products/read"},
		{Name: "ListProducts", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/products/read"},
		{Name: "DeleteProduct", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/products/delete"},
		{Name: "CreateOrUpdateProductAPI", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/products/apis/write"},
		{Name: "GetProductAPI", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/products/apis/read"},
		{Name: "ListProductAPIs", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/products/apis/read"},
		{Name: "DeleteProductAPI", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/products/apis/delete"},
		{Name: "CreateOrUpdateSubscription", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/subscriptions/write"},
		{Name: "GetSubscription", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/subscriptions/read"},
		{Name: "ListSubscriptions", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/subscriptions/read"},
		{Name: "DeleteSubscription", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/subscriptions/delete"},
		{Name: "CreateOrUpdateNamedValue", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/namedValues/write"},
		{Name: "GetNamedValue", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/namedValues/read"},
		{Name: "ListNamedValues", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/namedValues/read"},
		{Name: "DeleteNamedValue", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/namedValues/delete"},
		{Name: "CreateOrUpdateBackend", Method: http.MethodPut, IAMAction: "azure:Microsoft.ApiManagement/service/backends/write"},
		{Name: "GetBackend", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/backends/read"},
		{Name: "ListBackends", Method: http.MethodGet, IAMAction: "azure:Microsoft.ApiManagement/service/backends/read"},
		{Name: "DeleteBackend", Method: http.MethodDelete, IAMAction: "azure:Microsoft.ApiManagement/service/backends/delete"},
	}
}

func (s *APIManagementService) HealthCheck() error { return nil }

func (s *APIManagementService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(apiManagementAPIVersions))
	for _, apiVersion := range apiManagementAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.ApiManagement/service",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *APIManagementService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.ApiManagement/service")
}

func (s *APIManagementService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported API Management template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("API Management template resource is missing name")
	}
	body := map[string]any{
		"location":   resource["location"],
		"sku":        resource["sku"],
		"tags":       resource["tags"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateService(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *APIManagementService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}
	if serviceName, gatewayPath, ok := gatewayServiceAndPath(ctx.RawRequest); ok {
		return s.handleGatewayRequest(ctx, serviceName, gatewayPath)
	}
	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "service") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management route is not implemented.")
	}
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listServices(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.ChildType != "" {
		return s.handleChildRequest(ctx, route)
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateService(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getService(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteService(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handleChildRequest(ctx *service.RequestContext, route apiManagementRoute) (*service.Response, error) {
	if strings.EqualFold(route.ChildType, "policies") {
		return s.handlePolicyRequest(
			ctx,
			serviceKey(route.SubscriptionID, route.ResourceGroup, route.Name),
			serviceID(route.SubscriptionID, route.ResourceGroup, route.Name),
			"Microsoft.ApiManagement/service/policies",
			route.ChildName,
		)
	}
	if strings.EqualFold(route.ChildType, "products") {
		return s.handleProductRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "subscriptions") {
		return s.handleSubscriptionRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "namedValues") {
		return s.handleNamedValueRequest(ctx, route)
	}
	if strings.EqualFold(route.ChildType, "backends") {
		return s.handleBackendRequest(ctx, route)
	}
	if !strings.EqualFold(route.ChildType, "apis") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management child route is not implemented.")
	}
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listAPIs(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.GrandChildType == "" {
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createOrUpdateAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
		case http.MethodGet:
			return s.getAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		case http.MethodDelete:
			return s.deleteAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if strings.EqualFold(route.GrandChildType, "policies") {
		return s.handlePolicyRequest(
			ctx,
			apiKey(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName),
			serviceID(route.SubscriptionID, route.ResourceGroup, route.Name)+"/apis/"+route.ChildName,
			"Microsoft.ApiManagement/service/apis/policies",
			route.GrandChildName,
		)
	}
	if !strings.EqualFold(route.GrandChildType, "operations") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management API child route is not implemented.")
	}
	if route.GrandChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listOperations(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.GreatGrandChildType, "policies") {
		return s.handlePolicyRequest(
			ctx,
			operationKey(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName),
			serviceID(route.SubscriptionID, route.ResourceGroup, route.Name)+"/apis/"+route.ChildName+"/operations/"+route.GrandChildName,
			"Microsoft.ApiManagement/service/apis/operations/policies",
			route.GreatGrandChildName,
		)
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName, ctx.Body)
	case http.MethodGet:
		return s.getOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName)
	case http.MethodDelete:
		return s.deleteOperation(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handlePolicyRequest(ctx *service.RequestContext, parentKey, parentID, resourceType, policyID string) (*service.Response, error) {
	if !s.parentExists(parentKey, resourceType) {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", "The API Management policy parent resource could not be found.")
	}
	if policyID == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listPolicies(parentKey)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdatePolicy(parentKey, parentID, resourceType, policyID, ctx.Body)
	case http.MethodGet:
		return s.getPolicy(parentKey, policyID)
	case http.MethodDelete:
		return s.deletePolicy(parentKey, policyID)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handleProductRequest(ctx *service.RequestContext, route apiManagementRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listProducts(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if strings.EqualFold(route.GrandChildType, "apis") {
		if route.GrandChildName == "" {
			if ctx.RawRequest.Method == http.MethodGet {
				return s.listProductAPIs(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
			}
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		switch ctx.RawRequest.Method {
		case http.MethodPut:
			return s.createOrUpdateProductAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName)
		case http.MethodGet:
			return s.getProductAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName)
		case http.MethodDelete:
			return s.deleteProductAPI(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, route.GrandChildName)
		default:
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
	}
	if route.GrandChildType != "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management product child route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateProduct(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getProduct(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteProduct(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handleSubscriptionRequest(ctx *service.RequestContext, route apiManagementRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSubscriptions(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.GrandChildType != "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management subscription child route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSubscription(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getSubscription(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteSubscription(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handleNamedValueRequest(ctx *service.RequestContext, route apiManagementRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNamedValues(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.GrandChildType != "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management named value child route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateNamedValue(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getNamedValue(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteNamedValue(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *APIManagementService) handleBackendRequest(ctx *service.RequestContext, route apiManagementRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listBackends(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	if route.GrandChildType != "" {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The API Management backend child route is not implemented.")
	}
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateBackend(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getBackend(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteBackend(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

type apiManagementGatewayMatch struct {
	apiKey  string
	api     map[string]any
	apiID   string
	apiPath string
	suffix  string
}

func (s *APIManagementService) handleGatewayRequest(ctx *service.RequestContext, serviceName, gatewayPath string) (*service.Response, error) {
	s.mu.RLock()
	if !s.gatewayServiceExistsLocked(serviceName) {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("APIM service not found: %s", serviceName))
	}

	match, ok := s.matchGatewayAPILocked(serviceName, gatewayPath)
	if !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("No API route matched: %s", gatewayPath))
	}

	if message := s.gatewaySubscriptionFailureLocked(ctx.RawRequest, serviceName, match.apiKey); message != "" {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusUnauthorized, "Unauthorized", message)
	}

	operationID, hasOperations, operationMatched := s.matchGatewayOperationLocked(serviceName, match.apiKey, ctx.RawRequest.Method, match.suffix)
	if hasOperations && !operationMatched {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("No API operation matched: %s", gatewayPath))
	}

	queryParams := singleValueQuery(ctx.RawRequest.URL.Query())
	policy := s.applyGatewayPoliciesLocked(serviceName, match.apiID, operationID, match.suffix, queryParams)
	backendURL := firstNonBlank(policy.backendURL, stringValue(mapValue(match.api["properties"])["serviceUrl"]))
	mockBody := map[string]any{
		"service":     serviceName,
		"apiId":       match.apiID,
		"method":      ctx.RawRequest.Method,
		"path":        "/" + trimSlashes(gatewayPath),
		"backendPath": "/" + trimSlashes(policy.suffix),
		"operationId": operationID,
		"headers":     policy.headers,
		"queryParams": policy.queryParams,
	}
	s.mu.RUnlock()

	if policy.returnStatusCode != 0 {
		return policy.response()
	}
	if backendURL != "" {
		return s.proxyGatewayRequest(ctx, backendURL, policy.suffix, policy.headers, policy.queryParams)
	}

	return azurearm.JSONResponse(http.StatusOK, mockBody)
}

func (s *APIManagementService) gatewayServiceExistsLocked(serviceName string) bool {
	for _, resource := range s.services {
		if strings.EqualFold(resource.Name, serviceName) {
			return true
		}
	}
	return false
}

func (s *APIManagementService) matchGatewayAPILocked(serviceName, gatewayPath string) (apiManagementGatewayMatch, bool) {
	var best apiManagementGatewayMatch
	for key, api := range s.apis {
		if !apiResourceBelongsToService(api, serviceName) {
			continue
		}
		properties := mapValue(api["properties"])
		apiPath := trimSlashes(stringValue(properties["path"]))
		if !gatewayRouteMatches(apiPath, gatewayPath) {
			continue
		}
		match := apiManagementGatewayMatch{
			apiKey:  key,
			api:     api,
			apiID:   stringValue(api["name"]),
			apiPath: apiPath,
			suffix:  gatewayRouteSuffix(apiPath, gatewayPath),
		}
		if best.api == nil || len(match.apiPath) > len(best.apiPath) {
			best = match
		}
	}
	if best.api == nil {
		return apiManagementGatewayMatch{}, false
	}
	return best, true
}

func (s *APIManagementService) gatewaySubscriptionFailureLocked(req *http.Request, serviceName, apiKey string) string {
	requiredProducts := s.gatewayProductIDsForAPILocked(serviceName, apiKey)
	if len(requiredProducts) == 0 {
		return ""
	}

	key := firstNonBlank(req.Header.Get("Ocp-Apim-Subscription-Key"), req.URL.Query().Get("subscription-key"))
	if key == "" {
		return "Subscription key is required."
	}
	for _, subscription := range s.subscriptions {
		if !apiResourceBelongsToService(subscription, serviceName) {
			continue
		}
		properties := mapValue(subscription["properties"])
		if !strings.EqualFold(fmt.Sprint(properties["state"]), "active") {
			continue
		}
		if !subscriptionScopeMatches(requiredProducts, stringValue(properties["scope"])) {
			continue
		}
		if key == stringValue(properties["primaryKey"]) || key == stringValue(properties["secondaryKey"]) {
			return ""
		}
	}
	return "Subscription key is invalid."
}

func (s *APIManagementService) gatewayProductIDsForAPILocked(serviceName, apiKey string) []string {
	productIDs := make([]string, 0)
	for productAPIKey, linkedAPIKey := range s.productAPIs {
		if linkedAPIKey != apiKey {
			continue
		}
		if !strings.Contains(productAPIKey, "/"+strings.ToLower(serviceName)+"/products/") {
			continue
		}
		if productID := productIDFromProductAPIKey(productAPIKey); productID != "" {
			productIDs = append(productIDs, productID)
		}
	}
	sort.Strings(productIDs)
	return productIDs
}

func (s *APIManagementService) matchGatewayOperationLocked(serviceName, apiKey, method, suffix string) (string, bool, bool) {
	prefix := apiKey + "/operations/"
	hasOperations := false
	for key, operation := range s.operations {
		if !strings.HasPrefix(key, prefix) || !apiResourceBelongsToService(operation, serviceName) {
			continue
		}
		hasOperations = true
		properties := mapValue(operation["properties"])
		if !strings.EqualFold(method, stringValue(properties["method"])) {
			continue
		}
		if operationTemplateMatches(stringValue(properties["urlTemplate"]), "/"+trimSlashes(suffix)) {
			return stringValue(operation["name"]), true, true
		}
	}
	return "", hasOperations, false
}

type gatewayPolicyState struct {
	serviceName       string
	suffix            string
	backendURL        string
	headers           map[string]string
	queryParams       map[string]string
	returnStatusCode  int
	returnBody        string
	returnContentType string
}

func (state gatewayPolicyState) response() (*service.Response, error) {
	contentType := state.returnContentType
	if contentType == "" {
		trimmed := strings.TrimSpace(state.returnBody)
		if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
			contentType = "application/json"
		} else {
			contentType = "text/plain"
		}
	}
	return &service.Response{
		StatusCode:     state.returnStatusCode,
		Headers:        state.headers,
		RawBody:        []byte(state.returnBody),
		RawContentType: contentType,
	}, nil
}

func (s *APIManagementService) applyGatewayPoliciesLocked(serviceName, apiID, operationID, suffix string, queryParams map[string]string) gatewayPolicyState {
	state := gatewayPolicyState{
		serviceName: serviceName,
		suffix:      suffix,
		headers:     make(map[string]string),
		queryParams: queryParams,
	}
	for _, policy := range s.gatewayPoliciesLocked(serviceName, apiID, operationID) {
		value := stringValue(mapValue(policy["properties"])["value"])
		if strings.TrimSpace(value) != "" {
			s.applyGatewayPolicyXMLLocked(&state, value)
		}
	}
	return state
}

func (s *APIManagementService) gatewayPoliciesLocked(serviceName, apiID, operationID string) []map[string]any {
	policies := make([]map[string]any, 0, 3)
	if policy := s.gatewayPolicyByScopeLocked("Microsoft.ApiManagement/service/policies", serviceName, "", ""); policy != nil {
		policies = append(policies, policy)
	}
	if policy := s.gatewayPolicyByScopeLocked("Microsoft.ApiManagement/service/apis/policies", serviceName, apiID, ""); policy != nil {
		policies = append(policies, policy)
	}
	if operationID != "" {
		if policy := s.gatewayPolicyByScopeLocked("Microsoft.ApiManagement/service/apis/operations/policies", serviceName, apiID, operationID); policy != nil {
			policies = append(policies, policy)
		}
	}
	return policies
}

func (s *APIManagementService) gatewayPolicyByScopeLocked(resourceType, serviceName, apiID, operationID string) map[string]any {
	for _, policy := range s.policies {
		if policy["type"] != resourceType {
			continue
		}
		id := strings.ToLower(stringValue(policy["id"]))
		serviceSegment := "/service/" + strings.ToLower(serviceName)
		if !strings.Contains(id, serviceSegment) {
			continue
		}
		switch resourceType {
		case "Microsoft.ApiManagement/service/policies":
			if strings.Contains(id, serviceSegment+"/policies/") {
				return policy
			}
		case "Microsoft.ApiManagement/service/apis/policies":
			apiSegment := serviceSegment + "/apis/" + strings.ToLower(apiID)
			if strings.Contains(id, apiSegment+"/policies/") {
				return policy
			}
		case "Microsoft.ApiManagement/service/apis/operations/policies":
			operationSegment := serviceSegment + "/apis/" + strings.ToLower(apiID) + "/operations/" + strings.ToLower(operationID)
			if strings.Contains(id, operationSegment+"/policies/") {
				return policy
			}
		}
	}
	return nil
}

func (s *APIManagementService) applyGatewayPolicyXMLLocked(state *gatewayPolicyState, xmlValue string) {
	decoder := xml.NewDecoder(strings.NewReader(xmlValue))
	s.applyGatewayPolicyElementsLocked(state, decoder, "")
}

func (s *APIManagementService) applyGatewayPolicyElementsLocked(state *gatewayPolicyState, decoder *xml.Decoder, until string) {
	for {
		token, err := decoder.Token()
		if err != nil {
			return
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "set-backend-service":
				s.applyGatewaySetBackendServiceLocked(state, typed)
				skipXMLElement(decoder, typed.Name.Local)
			case "rewrite-uri":
				template := xmlAttr(typed, "template")
				if template != "" {
					state.suffix = trimSlashes(s.resolveGatewayNamedValuesLocked(state.serviceName, template))
				}
				skipXMLElement(decoder, typed.Name.Local)
			case "set-header":
				s.applyGatewaySetHeaderLocked(state, typed, readLastXMLChildText(decoder, typed.Name.Local, "value"))
			case "set-query-parameter":
				s.applyGatewaySetQueryParameterLocked(state, typed, readLastXMLChildText(decoder, typed.Name.Local, "value"))
			case "return-response":
				s.applyGatewayReturnResponseLocked(state, decoder)
			default:
				s.applyGatewayPolicyElementsLocked(state, decoder, typed.Name.Local)
			}
		case xml.EndElement:
			if until != "" && typed.Name.Local == until {
				return
			}
		}
	}
}

func (s *APIManagementService) applyGatewaySetBackendServiceLocked(state *gatewayPolicyState, start xml.StartElement) {
	if backendID := s.resolveGatewayNamedValuesLocked(state.serviceName, xmlAttr(start, "backend-id")); backendID != "" {
		if backendURL := s.gatewayBackendURLLocked(state.serviceName, backendID); backendURL != "" {
			state.backendURL = backendURL
		}
		return
	}
	if baseURL := s.resolveGatewayNamedValuesLocked(state.serviceName, xmlAttr(start, "base-url")); baseURL != "" {
		state.backendURL = baseURL
	}
}

func (s *APIManagementService) applyGatewaySetHeaderLocked(state *gatewayPolicyState, start xml.StartElement, value string) {
	name := xmlAttr(start, "name")
	if name == "" {
		return
	}
	action := xmlAttr(start, "exists-action")
	switch {
	case strings.EqualFold(action, "delete"):
		delete(state.headers, name)
	case strings.EqualFold(action, "skip"):
		if _, ok := state.headers[name]; !ok {
			state.headers[name] = s.resolveGatewayNamedValuesLocked(state.serviceName, value)
		}
	case strings.EqualFold(action, "append") && state.headers[name] != "":
		state.headers[name] += "," + s.resolveGatewayNamedValuesLocked(state.serviceName, value)
	default:
		state.headers[name] = s.resolveGatewayNamedValuesLocked(state.serviceName, value)
	}
}

func (s *APIManagementService) applyGatewaySetQueryParameterLocked(state *gatewayPolicyState, start xml.StartElement, value string) {
	name := xmlAttr(start, "name")
	if name == "" {
		return
	}
	action := xmlAttr(start, "exists-action")
	switch {
	case strings.EqualFold(action, "delete"):
		delete(state.queryParams, name)
	case strings.EqualFold(action, "skip"):
		if _, ok := state.queryParams[name]; !ok {
			state.queryParams[name] = s.resolveGatewayNamedValuesLocked(state.serviceName, value)
		}
	default:
		state.queryParams[name] = s.resolveGatewayNamedValuesLocked(state.serviceName, value)
	}
}

func (s *APIManagementService) applyGatewayReturnResponseLocked(state *gatewayPolicyState, decoder *xml.Decoder) {
	if state.returnStatusCode == 0 {
		state.returnStatusCode = http.StatusOK
	}
	for {
		token, err := decoder.Token()
		if err != nil {
			return
		}
		switch typed := token.(type) {
		case xml.StartElement:
			switch typed.Name.Local {
			case "set-status":
				if code, err := strconv.Atoi(xmlAttr(typed, "code")); err == nil && code > 0 {
					state.returnStatusCode = code
				}
				skipXMLElement(decoder, typed.Name.Local)
			case "set-header":
				s.applyGatewaySetHeaderLocked(state, typed, readLastXMLChildText(decoder, typed.Name.Local, "value"))
			case "set-body":
				state.returnBody = s.resolveGatewayNamedValuesLocked(state.serviceName, readXMLText(decoder, typed.Name.Local))
			default:
				skipXMLElement(decoder, typed.Name.Local)
			}
		case xml.EndElement:
			if typed.Name.Local == "return-response" {
				return
			}
		}
	}
}

func (s *APIManagementService) resolveGatewayNamedValuesLocked(serviceName, value string) string {
	if !strings.Contains(value, "{{") {
		return value
	}
	var out strings.Builder
	rest := value
	for {
		start := strings.Index(rest, "{{")
		if start < 0 {
			out.WriteString(rest)
			break
		}
		out.WriteString(rest[:start])
		afterStart := rest[start+2:]
		end := strings.Index(afterStart, "}}")
		if end < 0 {
			out.WriteString(rest[start:])
			break
		}
		name := strings.TrimSpace(afterStart[:end])
		replacement := s.gatewayNamedValueLocked(serviceName, name)
		if replacement == "" {
			replacement = "{{" + name + "}}"
		}
		out.WriteString(replacement)
		rest = afterStart[end+2:]
	}
	return out.String()
}

func (s *APIManagementService) gatewayNamedValueLocked(serviceName, name string) string {
	for _, namedValue := range s.namedValues {
		if !apiResourceBelongsToService(namedValue, serviceName) || !strings.EqualFold(stringValue(namedValue["name"]), name) {
			continue
		}
		return stringValue(mapValue(namedValue["properties"])["value"])
	}
	return ""
}

func (s *APIManagementService) proxyGatewayRequest(ctx *service.RequestContext, backendURL, suffix string, headers, queryParams map[string]string) (*service.Response, error) {
	target, err := gatewayProxyURL(backendURL, suffix, queryParams)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadGateway, "BackendUnavailable", err.Error())
	}
	req, err := http.NewRequestWithContext(ctx.RawRequest.Context(), ctx.RawRequest.Method, target, bytes.NewReader(ctx.Body))
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadGateway, "BackendUnavailable", err.Error())
	}
	if contentType := ctx.RawRequest.Header.Get("Content-Type"); contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	client := s.httpClient
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadGateway, "BackendUnavailable", err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadGateway, "BackendUnavailable", err.Error())
	}
	outHeaders := make(map[string]string)
	for key, values := range resp.Header {
		if len(values) > 0 && !strings.EqualFold(key, "Content-Type") {
			outHeaders[key] = values[0]
		}
	}
	return &service.Response{
		StatusCode:     resp.StatusCode,
		Headers:        outHeaders,
		RawBody:        data,
		RawContentType: resp.Header.Get("Content-Type"),
	}, nil
}

func gatewayProxyURL(baseURL, suffix string, queryParams map[string]string) (string, error) {
	target := strings.TrimRight(baseURL, "/")
	if trimmedSuffix := trimSlashes(suffix); trimmedSuffix != "" {
		target += "/" + trimmedSuffix
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	query := parsed.Query()
	for key, value := range queryParams {
		query.Set(key, value)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (s *APIManagementService) gatewayBackendURLLocked(serviceName, backendID string) string {
	for _, backend := range s.backends {
		if !apiResourceBelongsToService(backend, serviceName) || !strings.EqualFold(stringValue(backend["name"]), backendID) {
			continue
		}
		return stringValue(mapValue(backend["properties"])["url"])
	}
	return ""
}

func (s *APIManagementService) parentExists(parentKey, resourceType string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	switch resourceType {
	case "Microsoft.ApiManagement/service/policies":
		_, ok := s.services[parentKey]
		return ok
	case "Microsoft.ApiManagement/service/apis/policies":
		_, ok := s.apis[parentKey]
		return ok
	case "Microsoft.ApiManagement/service/apis/operations/policies":
		_, ok := s.operations[parentKey]
		return ok
	default:
		return false
	}
}

func (s *APIManagementService) createOrUpdateService(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
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
	if input.SKU == nil {
		input.SKU = map[string]any{"name": "Developer", "capacity": 1}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["gatewayUrl"]; !ok {
		input.Properties["gatewayUrl"] = gatewayURL(name)
	}
	if _, ok := input.Properties["managementApiUrl"]; !ok {
		input.Properties["managementApiUrl"] = "https://management.azure.com" + serviceID(subscriptionID, resourceGroup, name)
	}
	if _, ok := input.Properties["publisherEmail"]; !ok {
		input.Properties["publisherEmail"] = "admin@example.com"
	}
	if _, ok := input.Properties["publisherName"]; !ok {
		input.Properties["publisherName"] = "cloudmock"
	}

	resource := ServiceResource{
		ID:             serviceID(subscriptionID, resourceGroup, name),
		Name:           name,
		Type:           "Microsoft.ApiManagement/service",
		Location:       input.Location,
		Tags:           stringifyTags(input.Tags),
		SKU:            input.SKU,
		Properties:     input.Properties,
		SubscriptionID: subscriptionID,
		ResourceGroup:  resourceGroup,
	}

	s.mu.Lock()
	key := serviceKey(subscriptionID, resourceGroup, name)
	_, existed := s.services[key]
	s.services[key] = resource
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, resource)
}

func (s *APIManagementService) getService(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.services[serviceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *APIManagementService) listServices(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/"
	if resourceGroup != "" {
		prefix += strings.ToLower(resourceGroup) + "/"
	}
	values := make([]ServiceResource, 0)
	s.mu.RLock()
	for key, resource := range s.services {
		if strings.HasPrefix(key, prefix) {
			values = append(values, resource)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteService(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := serviceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.services[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", name))
	}
	delete(s.services, key)
	apiPrefix := key + "/apis/"
	for apiKey := range s.apis {
		if strings.HasPrefix(apiKey, apiPrefix) {
			delete(s.apis, apiKey)
		}
	}
	for operationKey := range s.operations {
		if strings.HasPrefix(operationKey, apiPrefix) {
			delete(s.operations, operationKey)
		}
	}
	for policyKey := range s.policies {
		if strings.HasPrefix(policyKey, key) {
			delete(s.policies, policyKey)
		}
	}
	for productKey := range s.products {
		if strings.HasPrefix(productKey, key) {
			delete(s.products, productKey)
		}
	}
	for productAPIKey := range s.productAPIs {
		if strings.HasPrefix(productAPIKey, key) {
			delete(s.productAPIs, productAPIKey)
		}
	}
	for subscriptionKey := range s.subscriptions {
		if strings.HasPrefix(subscriptionKey, key) {
			delete(s.subscriptions, subscriptionKey)
		}
	}
	for namedValueKey := range s.namedValues {
		if strings.HasPrefix(namedValueKey, key) {
			delete(s.namedValues, namedValueKey)
		}
	}
	for backendKey := range s.backends {
		if strings.HasPrefix(backendKey, key) {
			delete(s.backends, backendKey)
		}
	}
	return &service.Response{StatusCode: http.StatusAccepted, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateAPI(subscriptionID, resourceGroup, serviceName, apiID string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["displayName"]; !ok {
		input.Properties["displayName"] = apiID
	}
	if _, ok := input.Properties["path"]; !ok {
		input.Properties["path"] = apiID
	}
	if _, ok := input.Properties["protocols"]; !ok {
		input.Properties["protocols"] = []string{"https"}
	}
	input.Properties["provisioningState"] = "Succeeded"

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	key := apiKey(subscriptionID, resourceGroup, serviceName, apiID)
	_, existed := s.apis[key]
	resource := apiResource(subscriptionID, resourceGroup, serviceName, apiID, input.Properties)
	s.apis[key] = resource
	s.importOpenAPIOperationsLocked(subscriptionID, resourceGroup, serviceName, apiID, input.Properties)

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, resource)
}

func (s *APIManagementService) importOpenAPIOperationsLocked(subscriptionID, resourceGroup, serviceName, apiID string, apiProperties map[string]any) {
	format := strings.ToLower(stringValue(apiProperties["format"]))
	if !strings.Contains(format, "openapi") {
		return
	}

	document, ok := openAPIDocument(apiProperties["value"])
	if !ok {
		return
	}
	paths := mapValue(document["paths"])
	if paths == nil {
		return
	}

	prefix := operationPrefix(subscriptionID, resourceGroup, serviceName, apiID)
	for key := range s.operations {
		if strings.HasPrefix(key, prefix) {
			delete(s.operations, key)
		}
	}

	pathNames := make([]string, 0, len(paths))
	for path := range paths {
		pathNames = append(pathNames, path)
	}
	sort.Strings(pathNames)
	for _, path := range pathNames {
		s.importOpenAPIPathLocked(subscriptionID, resourceGroup, serviceName, apiID, path, mapValue(paths[path]))
	}
}

func (s *APIManagementService) importOpenAPIPathLocked(subscriptionID, resourceGroup, serviceName, apiID, path string, pathItem map[string]any) {
	if pathItem == nil {
		return
	}

	methodNames := make([]string, 0, len(pathItem))
	for method := range pathItem {
		methodNames = append(methodNames, method)
	}
	sort.Strings(methodNames)
	for _, methodName := range methodNames {
		method := strings.ToUpper(methodName)
		if !isOpenAPIHTTPMethod(method) {
			continue
		}
		operation := mapValue(pathItem[methodName])
		if operation == nil {
			continue
		}
		operationID := firstNonBlank(stringValue(operation["operationId"]), generatedOpenAPIOperationID(method, path))
		properties := map[string]any{
			"displayName": firstNonBlank(stringValue(operation["summary"]), operationID),
			"method":      method,
			"urlTemplate": firstNonBlank(path, "/"),
		}
		s.operations[operationKey(subscriptionID, resourceGroup, serviceName, apiID, operationID)] = operationResource(subscriptionID, resourceGroup, serviceName, apiID, operationID, properties)
	}
}

func (s *APIManagementService) getAPI(subscriptionID, resourceGroup, serviceName, apiID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.apis[apiKey(subscriptionID, resourceGroup, serviceName, apiID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API %q could not be found.", apiID))
	}
	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *APIManagementService) listAPIs(subscriptionID, resourceGroup, serviceName string) (*service.Response, error) {
	s.mu.RLock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	prefix := apiPrefix(subscriptionID, resourceGroup, serviceName)
	values := make([]map[string]any, 0)
	for key, resource := range s.apis {
		if strings.HasPrefix(key, prefix) {
			values = append(values, resource)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteAPI(subscriptionID, resourceGroup, serviceName, apiID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := apiKey(subscriptionID, resourceGroup, serviceName, apiID)
	if _, ok := s.apis[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API %q could not be found.", apiID))
	}
	delete(s.apis, key)
	for operationKey := range s.operations {
		if strings.HasPrefix(operationKey, key+"/operations/") {
			delete(s.operations, operationKey)
		}
	}
	for policyKey := range s.policies {
		if strings.HasPrefix(policyKey, key) {
			delete(s.policies, policyKey)
		}
	}
	for productAPIKey, linkedAPIKey := range s.productAPIs {
		if linkedAPIKey == key {
			delete(s.productAPIs, productAPIKey)
		}
	}
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateOperation(subscriptionID, resourceGroup, serviceName, apiID, operationID string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["displayName"]; !ok {
		input.Properties["displayName"] = operationID
	}
	if _, ok := input.Properties["method"]; !ok {
		input.Properties["method"] = "GET"
	}
	if _, ok := input.Properties["urlTemplate"]; !ok {
		input.Properties["urlTemplate"] = "/"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.apis[apiKey(subscriptionID, resourceGroup, serviceName, apiID)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API %q could not be found.", apiID))
	}
	key := operationKey(subscriptionID, resourceGroup, serviceName, apiID, operationID)
	_, existed := s.operations[key]
	resource := operationResource(subscriptionID, resourceGroup, serviceName, apiID, operationID, input.Properties)
	s.operations[key] = resource

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, resource)
}

func (s *APIManagementService) getOperation(subscriptionID, resourceGroup, serviceName, apiID, operationID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.operations[operationKey(subscriptionID, resourceGroup, serviceName, apiID, operationID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Operation %q could not be found.", operationID))
	}
	return azurearm.JSONResponse(http.StatusOK, resource)
}

func (s *APIManagementService) listOperations(subscriptionID, resourceGroup, serviceName, apiID string) (*service.Response, error) {
	s.mu.RLock()
	if _, ok := s.apis[apiKey(subscriptionID, resourceGroup, serviceName, apiID)]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API %q could not be found.", apiID))
	}
	prefix := operationPrefix(subscriptionID, resourceGroup, serviceName, apiID)
	values := make([]map[string]any, 0)
	for key, resource := range s.operations {
		if strings.HasPrefix(key, prefix) {
			values = append(values, resource)
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteOperation(subscriptionID, resourceGroup, serviceName, apiID, operationID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := operationKey(subscriptionID, resourceGroup, serviceName, apiID, operationID)
	if _, ok := s.operations[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Operation %q could not be found.", operationID))
	}
	delete(s.operations, key)
	for policyKey := range s.policies {
		if strings.HasPrefix(policyKey, key) {
			delete(s.policies, policyKey)
		}
	}
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdatePolicy(parentKey, parentID, resourceType, policyID string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["format"]; !ok {
		input.Properties["format"] = "rawxml"
	}
	if _, ok := input.Properties["value"]; !ok {
		input.Properties["value"] = ""
	}

	resource := map[string]any{
		"_parent":    parentKey,
		"id":         parentID + "/policies/" + policyID,
		"name":       policyID,
		"type":       resourceType,
		"properties": input.Properties,
	}
	s.mu.Lock()
	s.policies[policyKey(parentKey, policyID)] = resource
	s.mu.Unlock()
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) getPolicy(parentKey, policyID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.policies[policyKey(parentKey, policyID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Policy %q could not be found.", policyID))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) listPolicies(parentKey string) (*service.Response, error) {
	values := make([]map[string]any, 0)
	s.mu.RLock()
	for _, resource := range s.policies {
		if resource["_parent"] == parentKey {
			values = append(values, stripInternal(resource))
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deletePolicy(parentKey, policyID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := policyKey(parentKey, policyID)
	if _, ok := s.policies[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Policy %q could not be found.", policyID))
	}
	delete(s.policies, key)
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateProduct(subscriptionID, resourceGroup, serviceName, productID string, body []byte) (*service.Response, error) {
	properties, err := requestProperties(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if _, ok := properties["displayName"]; !ok {
		properties["displayName"] = productID
	}
	if _, ok := properties["subscriptionRequired"]; !ok {
		properties["subscriptionRequired"] = true
	}
	if _, ok := properties["approvalRequired"]; !ok {
		properties["approvalRequired"] = false
	}
	if _, ok := properties["state"]; !ok {
		properties["state"] = "published"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	resource := childResource(subscriptionID, resourceGroup, serviceName, "products", productID, "Microsoft.ApiManagement/service/products", properties)
	resource["_service"] = serviceName
	s.products[productKey(subscriptionID, resourceGroup, serviceName, productID)] = resource
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) getProduct(subscriptionID, resourceGroup, serviceName, productID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.products[productKey(subscriptionID, resourceGroup, serviceName, productID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product %q could not be found.", productID))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) listProducts(subscriptionID, resourceGroup, serviceName string) (*service.Response, error) {
	svcKey := serviceKey(subscriptionID, resourceGroup, serviceName)
	prefix := productPrefix(subscriptionID, resourceGroup, serviceName)
	s.mu.RLock()
	if _, ok := s.services[svcKey]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	values := sortedResourcesWithPrefix(s.products, prefix)
	s.mu.RUnlock()
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteProduct(subscriptionID, resourceGroup, serviceName, productID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := productKey(subscriptionID, resourceGroup, serviceName, productID)
	if _, ok := s.products[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product %q could not be found.", productID))
	}
	delete(s.products, key)
	linkPrefix := key + "/apis/"
	for linkKey := range s.productAPIs {
		if strings.HasPrefix(linkKey, linkPrefix) {
			delete(s.productAPIs, linkKey)
		}
	}
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateProductAPI(subscriptionID, resourceGroup, serviceName, productID, apiID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	productKey := productKey(subscriptionID, resourceGroup, serviceName, productID)
	if _, ok := s.products[productKey]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product %q could not be found.", productID))
	}
	apiKey := apiKey(subscriptionID, resourceGroup, serviceName, apiID)
	api, ok := s.apis[apiKey]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API %q could not be found.", apiID))
	}
	s.productAPIs[productAPIKey(subscriptionID, resourceGroup, serviceName, productID, apiID)] = apiKey
	return azurearm.JSONResponse(http.StatusOK, stripInternal(api))
}

func (s *APIManagementService) getProductAPI(subscriptionID, resourceGroup, serviceName, productID, apiID string) (*service.Response, error) {
	s.mu.RLock()
	apiKey := s.productAPIs[productAPIKey(subscriptionID, resourceGroup, serviceName, productID, apiID)]
	api, ok := s.apis[apiKey]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product API link %q/%q could not be found.", productID, apiID))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(api))
}

func (s *APIManagementService) listProductAPIs(subscriptionID, resourceGroup, serviceName, productID string) (*service.Response, error) {
	productKey := productKey(subscriptionID, resourceGroup, serviceName, productID)
	linkPrefix := productKey + "/apis/"
	values := make([]map[string]any, 0)
	s.mu.RLock()
	if _, ok := s.products[productKey]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product %q could not be found.", productID))
	}
	for linkKey, apiKey := range s.productAPIs {
		if strings.HasPrefix(linkKey, linkPrefix) {
			if api, ok := s.apis[apiKey]; ok {
				values = append(values, stripInternal(api))
			}
		}
	}
	s.mu.RUnlock()
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteProductAPI(subscriptionID, resourceGroup, serviceName, productID, apiID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := productAPIKey(subscriptionID, resourceGroup, serviceName, productID, apiID)
	if _, ok := s.productAPIs[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Product API link %q/%q could not be found.", productID, apiID))
	}
	delete(s.productAPIs, key)
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateSubscription(subscriptionID, resourceGroup, serviceName, subscriptionIDResource string, body []byte) (*service.Response, error) {
	properties, err := requestProperties(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if _, ok := properties["displayName"]; !ok {
		properties["displayName"] = subscriptionIDResource
	}
	if _, ok := properties["state"]; !ok {
		properties["state"] = "active"
	}
	if _, ok := properties["scope"]; !ok {
		properties["scope"] = serviceID(subscriptionID, resourceGroup, serviceName)
	}
	if _, ok := properties["primaryKey"]; !ok {
		properties["primaryKey"] = generatedSubscriptionKey(subscriptionIDResource, "primary")
	}
	if _, ok := properties["secondaryKey"]; !ok {
		properties["secondaryKey"] = generatedSubscriptionKey(subscriptionIDResource, "secondary")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	resource := childResource(subscriptionID, resourceGroup, serviceName, "subscriptions", subscriptionIDResource, "Microsoft.ApiManagement/service/subscriptions", properties)
	resource["_service"] = serviceName
	s.subscriptions[subscriptionKey(subscriptionID, resourceGroup, serviceName, subscriptionIDResource)] = resource
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) getSubscription(subscriptionID, resourceGroup, serviceName, subscriptionIDResource string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.subscriptions[subscriptionKey(subscriptionID, resourceGroup, serviceName, subscriptionIDResource)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Subscription %q could not be found.", subscriptionIDResource))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) listSubscriptions(subscriptionID, resourceGroup, serviceName string) (*service.Response, error) {
	svcKey := serviceKey(subscriptionID, resourceGroup, serviceName)
	prefix := subscriptionPrefix(subscriptionID, resourceGroup, serviceName)
	s.mu.RLock()
	if _, ok := s.services[svcKey]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	values := sortedResourcesWithPrefix(s.subscriptions, prefix)
	s.mu.RUnlock()
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteSubscription(subscriptionID, resourceGroup, serviceName, subscriptionIDResource string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := subscriptionKey(subscriptionID, resourceGroup, serviceName, subscriptionIDResource)
	if _, ok := s.subscriptions[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Subscription %q could not be found.", subscriptionIDResource))
	}
	delete(s.subscriptions, key)
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateNamedValue(subscriptionID, resourceGroup, serviceName, namedValueID string, body []byte) (*service.Response, error) {
	properties, err := requestProperties(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if _, ok := properties["displayName"]; !ok {
		properties["displayName"] = namedValueID
	}
	if _, ok := properties["value"]; !ok {
		properties["value"] = ""
	}
	if _, ok := properties["secret"]; !ok {
		properties["secret"] = false
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	resource := childResource(subscriptionID, resourceGroup, serviceName, "namedValues", namedValueID, "Microsoft.ApiManagement/service/namedValues", properties)
	resource["_service"] = serviceName
	s.namedValues[namedValueKey(subscriptionID, resourceGroup, serviceName, namedValueID)] = resource
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) getNamedValue(subscriptionID, resourceGroup, serviceName, namedValueID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.namedValues[namedValueKey(subscriptionID, resourceGroup, serviceName, namedValueID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Named value %q could not be found.", namedValueID))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) listNamedValues(subscriptionID, resourceGroup, serviceName string) (*service.Response, error) {
	svcKey := serviceKey(subscriptionID, resourceGroup, serviceName)
	prefix := namedValuePrefix(subscriptionID, resourceGroup, serviceName)
	s.mu.RLock()
	if _, ok := s.services[svcKey]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	values := sortedResourcesWithPrefix(s.namedValues, prefix)
	s.mu.RUnlock()
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteNamedValue(subscriptionID, resourceGroup, serviceName, namedValueID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := namedValueKey(subscriptionID, resourceGroup, serviceName, namedValueID)
	if _, ok := s.namedValues[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Named value %q could not be found.", namedValueID))
	}
	delete(s.namedValues, key)
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

func (s *APIManagementService) createOrUpdateBackend(subscriptionID, resourceGroup, serviceName, backendID string, body []byte) (*service.Response, error) {
	properties, err := requestProperties(body)
	if err != nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
	}
	if _, ok := properties["title"]; !ok {
		properties["title"] = backendID
	}
	if _, ok := properties["protocol"]; !ok {
		properties["protocol"] = "http"
	}
	if _, ok := properties["url"]; !ok {
		properties["url"] = ""
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.services[serviceKey(subscriptionID, resourceGroup, serviceName)]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	resource := childResource(subscriptionID, resourceGroup, serviceName, "backends", backendID, "Microsoft.ApiManagement/service/backends", properties)
	resource["_service"] = serviceName
	s.backends[backendKey(subscriptionID, resourceGroup, serviceName, backendID)] = resource
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) getBackend(subscriptionID, resourceGroup, serviceName, backendID string) (*service.Response, error) {
	s.mu.RLock()
	resource, ok := s.backends[backendKey(subscriptionID, resourceGroup, serviceName, backendID)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Backend %q could not be found.", backendID))
	}
	return azurearm.JSONResponse(http.StatusOK, stripInternal(resource))
}

func (s *APIManagementService) listBackends(subscriptionID, resourceGroup, serviceName string) (*service.Response, error) {
	svcKey := serviceKey(subscriptionID, resourceGroup, serviceName)
	prefix := backendPrefix(subscriptionID, resourceGroup, serviceName)
	s.mu.RLock()
	if _, ok := s.services[svcKey]; !ok {
		s.mu.RUnlock()
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("API Management service %q could not be found.", serviceName))
	}
	values := sortedResourcesWithPrefix(s.backends, prefix)
	s.mu.RUnlock()
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *APIManagementService) deleteBackend(subscriptionID, resourceGroup, serviceName, backendID string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	key := backendKey(subscriptionID, resourceGroup, serviceName, backendID)
	if _, ok := s.backends[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Backend %q could not be found.", backendID))
	}
	delete(s.backends, key)
	return &service.Response{StatusCode: http.StatusOK, RawContentType: "application/json"}, nil
}

type apiManagementRoute struct {
	SubscriptionID      string
	ResourceGroup       string
	ResourceType        string
	Name                string
	ChildType           string
	ChildName           string
	GrandChildType      string
	GrandChildName      string
	GreatGrandChildType string
	GreatGrandChildName string
}

func parseRoute(escapedPath string) (apiManagementRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) == 5 &&
		strings.EqualFold(parts[0], "subscriptions") &&
		strings.EqualFold(parts[2], "providers") &&
		strings.EqualFold(parts[3], "Microsoft.ApiManagement") &&
		strings.EqualFold(parts[4], "service") {
		return apiManagementRoute{SubscriptionID: parts[1], ResourceType: "service"}, true
	}
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.ApiManagement") {
		return apiManagementRoute{}, false
	}
	route := apiManagementRoute{
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
		route.ChildType = parts[8]
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
		route.GrandChildType = parts[10]
		return route, true
	case 12:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		route.GrandChildName = parts[11]
		return route, true
	case 13:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		route.GrandChildName = parts[11]
		route.GreatGrandChildType = parts[12]
		return route, true
	case 14:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		route.GrandChildType = parts[10]
		route.GrandChildName = parts[11]
		route.GreatGrandChildType = parts[12]
		route.GreatGrandChildName = parts[13]
		return route, true
	default:
		return apiManagementRoute{}, false
	}
}

func apiResource(subscriptionID, resourceGroup, serviceName, apiID string, properties map[string]any) map[string]any {
	return map[string]any{
		"id":         serviceID(subscriptionID, resourceGroup, serviceName) + "/apis/" + apiID,
		"name":       apiID,
		"type":       "Microsoft.ApiManagement/service/apis",
		"properties": properties,
	}
}

func operationResource(subscriptionID, resourceGroup, serviceName, apiID, operationID string, properties map[string]any) map[string]any {
	return map[string]any{
		"id":         serviceID(subscriptionID, resourceGroup, serviceName) + "/apis/" + apiID + "/operations/" + operationID,
		"name":       operationID,
		"type":       "Microsoft.ApiManagement/service/apis/operations",
		"properties": properties,
	}
}

func childResource(subscriptionID, resourceGroup, serviceName, collection, name, resourceType string, properties map[string]any) map[string]any {
	return map[string]any{
		"id":         serviceID(subscriptionID, resourceGroup, serviceName) + "/" + collection + "/" + name,
		"name":       name,
		"type":       resourceType,
		"properties": properties,
	}
}

func serviceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ApiManagement/service/" + name
}

func serviceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func apiPrefix(subscriptionID, resourceGroup, serviceName string) string {
	return serviceKey(subscriptionID, resourceGroup, serviceName) + "/apis/"
}

func apiKey(subscriptionID, resourceGroup, serviceName, apiID string) string {
	return apiPrefix(subscriptionID, resourceGroup, serviceName) + strings.ToLower(apiID)
}

func operationPrefix(subscriptionID, resourceGroup, serviceName, apiID string) string {
	return apiKey(subscriptionID, resourceGroup, serviceName, apiID) + "/operations/"
}

func operationKey(subscriptionID, resourceGroup, serviceName, apiID, operationID string) string {
	return operationPrefix(subscriptionID, resourceGroup, serviceName, apiID) + strings.ToLower(operationID)
}

func policyKey(parentKey, policyID string) string {
	return parentKey + "/policies/" + strings.ToLower(policyID)
}

func productPrefix(subscriptionID, resourceGroup, serviceName string) string {
	return serviceKey(subscriptionID, resourceGroup, serviceName) + "/products/"
}

func productKey(subscriptionID, resourceGroup, serviceName, productID string) string {
	return productPrefix(subscriptionID, resourceGroup, serviceName) + strings.ToLower(productID)
}

func productAPIKey(subscriptionID, resourceGroup, serviceName, productID, apiID string) string {
	return productKey(subscriptionID, resourceGroup, serviceName, productID) + "/apis/" + strings.ToLower(apiID)
}

func subscriptionPrefix(subscriptionID, resourceGroup, serviceName string) string {
	return serviceKey(subscriptionID, resourceGroup, serviceName) + "/subscriptions/"
}

func subscriptionKey(subscriptionID, resourceGroup, serviceName, subscriptionIDResource string) string {
	return subscriptionPrefix(subscriptionID, resourceGroup, serviceName) + strings.ToLower(subscriptionIDResource)
}

func namedValuePrefix(subscriptionID, resourceGroup, serviceName string) string {
	return serviceKey(subscriptionID, resourceGroup, serviceName) + "/namedValues/"
}

func namedValueKey(subscriptionID, resourceGroup, serviceName, namedValueID string) string {
	return namedValuePrefix(subscriptionID, resourceGroup, serviceName) + strings.ToLower(namedValueID)
}

func backendPrefix(subscriptionID, resourceGroup, serviceName string) string {
	return serviceKey(subscriptionID, resourceGroup, serviceName) + "/backends/"
}

func backendKey(subscriptionID, resourceGroup, serviceName, backendID string) string {
	return backendPrefix(subscriptionID, resourceGroup, serviceName) + strings.ToLower(backendID)
}

func requestProperties(body []byte) (map[string]any, error) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return nil, err
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	return input.Properties, nil
}

func sortedResourcesWithPrefix(resources map[string]map[string]any, prefix string) []map[string]any {
	values := make([]map[string]any, 0)
	for key, resource := range resources {
		if strings.HasPrefix(key, prefix) {
			values = append(values, stripInternal(resource))
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i]["name"].(string) < values[j]["name"].(string) })
	return values
}

func stripInternal(resource map[string]any) map[string]any {
	out := make(map[string]any, len(resource))
	for key, value := range resource {
		if strings.HasPrefix(key, "_") {
			continue
		}
		out[key] = value
	}
	if out["type"] == "Microsoft.ApiManagement/service/namedValues" {
		properties, ok := out["properties"].(map[string]any)
		if ok && boolValue(properties["secret"]) {
			filtered := make(map[string]any, len(properties))
			for key, value := range properties {
				if key != "value" {
					filtered[key] = value
				}
			}
			out["properties"] = filtered
		}
	}
	return out
}

func generatedSubscriptionKey(subscriptionIDResource, suffix string) string {
	sum := sha256.Sum256([]byte(subscriptionIDResource + ":" + suffix))
	return hex.EncodeToString(sum[:])[:32]
}

func gatewayURL(name string) string {
	return "http://localhost:4577/devstoreaccount1-apim/" + name
}

func gatewayServiceAndPath(req *http.Request) (string, string, bool) {
	parts := splitPath(req.URL.EscapedPath())
	if len(parts) < 2 || !strings.HasSuffix(strings.ToLower(parts[0]), "-apim") {
		return "", "", false
	}
	return parts[1], strings.Join(parts[2:], "/"), true
}

func apiResourceBelongsToService(resource map[string]any, serviceName string) bool {
	id := strings.ToLower(stringValue(resource["id"]))
	return strings.Contains(id, "/service/"+strings.ToLower(serviceName)+"/")
}

func gatewayRouteMatches(apiPath, gatewayPath string) bool {
	api := trimSlashes(apiPath)
	path := trimSlashes(gatewayPath)
	return api == "" || path == api || strings.HasPrefix(path, api+"/")
}

func gatewayRouteSuffix(apiPath, gatewayPath string) string {
	api := trimSlashes(apiPath)
	path := trimSlashes(gatewayPath)
	if api == "" {
		return path
	}
	if path == api {
		return ""
	}
	return strings.TrimPrefix(path, api+"/")
}

func operationTemplateMatches(template, path string) bool {
	templateParts := splitCleanPath(template)
	pathParts := splitCleanPath(path)
	if len(templateParts) != len(pathParts) {
		return false
	}
	for i, templatePart := range templateParts {
		if strings.HasPrefix(templatePart, "{") && strings.HasSuffix(templatePart, "}") {
			continue
		}
		if templatePart != pathParts[i] {
			return false
		}
	}
	return true
}

func splitCleanPath(value string) []string {
	value = trimSlashes(value)
	if value == "" {
		return nil
	}
	return strings.Split(value, "/")
}

func trimSlashes(value string) string {
	return strings.Trim(value, "/")
}

func singleValueQuery(values url.Values) map[string]string {
	out := make(map[string]string, len(values))
	for key, value := range values {
		if len(value) > 0 {
			out[key] = value[0]
		}
	}
	return out
}

func mapValue(value any) map[string]any {
	if typed, ok := value.(map[string]any); ok {
		return typed
	}
	return nil
}

func firstNonBlank(first, second string) string {
	if strings.TrimSpace(first) != "" {
		return first
	}
	return second
}

func openAPIDocument(value any) (map[string]any, bool) {
	switch typed := value.(type) {
	case map[string]any:
		return typed, true
	case string:
		if strings.TrimSpace(typed) == "" {
			return nil, false
		}
		var document map[string]any
		if err := gojson.Unmarshal([]byte(typed), &document); err != nil {
			return nil, false
		}
		return document, true
	default:
		return nil, false
	}
}

func isOpenAPIHTTPMethod(method string) bool {
	switch strings.ToUpper(method) {
	case "GET", "PUT", "POST", "DELETE", "PATCH", "HEAD", "OPTIONS", "TRACE":
		return true
	default:
		return false
	}
}

func generatedOpenAPIOperationID(method, path string) string {
	normalized := trimSlashes(path)
	if normalized == "" {
		normalized = "root"
	}
	replacer := strings.NewReplacer("/", "-", "{", "", "}", "", "_", "-")
	return strings.ToLower(method) + "-" + replacer.Replace(normalized)
}

func productIDFromProductAPIKey(key string) string {
	parts := strings.Split(key, "/")
	for i, part := range parts {
		if part == "products" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}

func subscriptionScopeMatches(productIDs []string, scope string) bool {
	scope = strings.ToLower(strings.TrimSpace(scope))
	for _, productID := range productIDs {
		productScope := "/products/" + strings.ToLower(productID)
		if scope == productScope || strings.HasSuffix(scope, productScope) {
			return true
		}
	}
	return false
}

func xmlAttr(start xml.StartElement, name string) string {
	for _, attr := range start.Attr {
		if attr.Name.Local == name {
			return attr.Value
		}
	}
	return ""
}

func readLastXMLChildText(decoder *xml.Decoder, endName, childName string) string {
	depth := 1
	inChild := false
	var current strings.Builder
	last := ""
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch typed := token.(type) {
		case xml.StartElement:
			depth++
			if typed.Name.Local == childName {
				inChild = true
				current.Reset()
			}
		case xml.CharData:
			if inChild {
				current.Write([]byte(typed))
			}
		case xml.EndElement:
			if typed.Name.Local == childName && inChild {
				last = strings.TrimSpace(current.String())
				inChild = false
			}
			depth--
			if typed.Name.Local == endName && depth == 0 {
				return last
			}
		}
	}
	return last
}

func readXMLText(decoder *xml.Decoder, endName string) string {
	depth := 1
	var text strings.Builder
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}
		switch typed := token.(type) {
		case xml.StartElement:
			depth++
		case xml.CharData:
			text.Write([]byte(typed))
		case xml.EndElement:
			depth--
			if typed.Name.Local == endName && depth == 0 {
				return strings.TrimSpace(text.String())
			}
		}
	}
	return strings.TrimSpace(text.String())
}

func skipXMLElement(decoder *xml.Decoder, endName string) {
	depth := 1
	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			return
		}
		switch typed := token.(type) {
		case xml.StartElement:
			depth++
		case xml.EndElement:
			depth--
			if typed.Name.Local == endName && depth == 0 {
				return
			}
		}
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

func stringValue(value any) string {
	if typed, ok := value.(string); ok {
		return typed
	}
	return ""
}

func boolValue(value any) bool {
	switch typed := value.(type) {
	case bool:
		return typed
	case string:
		return strings.EqualFold(typed, "true")
	default:
		return false
	}
}
