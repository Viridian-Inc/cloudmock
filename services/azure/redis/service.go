package redis

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

var redisAPIVersions = []string{"2024-11-01", "2023-08-01"}

// RedisService implements first-slice Azure Cache for Redis control-plane APIs.
type RedisService struct {
	mu          sync.RWMutex
	caches      map[string]Cache
	keys        map[string]accessKeys
	keyCounters map[string]map[string]int
}

func New() *RedisService {
	return &RedisService{
		caches:      make(map[string]Cache),
		keys:        make(map[string]accessKeys),
		keyCounters: make(map[string]map[string]int),
	}
}

func (s *RedisService) Name() string { return "Microsoft.Cache" }

func (s *RedisService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.Cache/Redis/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cache/Redis/read"},
		{Name: "ListByResourceGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Cache/Redis/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Cache/Redis/delete"},
		{Name: "ListKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.Cache/Redis/listKeys/action"},
		{Name: "RegenerateKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.Cache/Redis/regenerateKey/action"},
	}
}

func (s *RedisService) HealthCheck() error { return nil }

func (s *RedisService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(redisAPIVersions))
	for _, apiVersion := range redisAPIVersions {
		keys = append(keys, routing.ServiceKey{
			Provider:   routing.ProviderAzure,
			Service:    "Microsoft.Cache/Redis",
			APIVersion: apiVersion,
		})
	}
	return keys
}

func (s *RedisService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.Cache/Redis")
}

func (s *RedisService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Redis template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Redis template resource is missing name")
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
	resp, err := s.createOrUpdateCache(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *RedisService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok || !strings.EqualFold(route.ResourceType, "Redis") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Redis route is not implemented.")
	}
	if route.Operation != "" {
		return s.handleOperation(ctx, route)
	}
	return s.handleCacheRequest(ctx, route)
}

func (s *RedisService) handleCacheRequest(ctx *service.RequestContext, route redisRoute) (*service.Response, error) {
	if route.CacheName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listCaches(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateCache(route.SubscriptionID, route.ResourceGroup, route.CacheName, ctx.Body)
	case http.MethodGet:
		return s.getCache(route.SubscriptionID, route.ResourceGroup, route.CacheName)
	case http.MethodDelete:
		return s.deleteCache(route.SubscriptionID, route.ResourceGroup, route.CacheName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *RedisService) handleOperation(ctx *service.RequestContext, route redisRoute) (*service.Response, error) {
	if ctx.RawRequest.Method != http.MethodPost {
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
	switch {
	case strings.EqualFold(route.Operation, "listKeys"):
		return s.listKeys(route.SubscriptionID, route.ResourceGroup, route.CacheName)
	case strings.EqualFold(route.Operation, "regenerateKey"):
		return s.regenerateKey(route.SubscriptionID, route.ResourceGroup, route.CacheName, ctx.Body)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Redis operation is not implemented.")
	}
}

func (s *RedisService) createOrUpdateCache(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
	if _, ok := input.Properties["hostName"]; !ok {
		input.Properties["hostName"] = name + ".redis.cache.windows.net"
	}
	if _, ok := input.Properties["sslPort"]; !ok {
		input.Properties["sslPort"] = 6380
	}
	if _, ok := input.Properties["port"]; !ok {
		input.Properties["port"] = 6379
	}

	cache := Cache{
		ID:         cacheID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Cache/Redis",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := cacheKey(subscriptionID, resourceGroup, name)
	_, existed := s.caches[key]
	s.caches[key] = cache
	if _, ok := s.keys[key]; !ok {
		s.keys[key] = initialKeys(name)
	}
	if s.keyCounters[key] == nil {
		s.keyCounters[key] = map[string]int{"primary": 0, "secondary": 0}
	}
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, cache)
}

func (s *RedisService) getCache(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	cache, ok := s.caches[cacheKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Redis cache %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, cache)
}

func (s *RedisService) listCaches(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]Cache, 0)
	for key, cache := range s.caches {
		if strings.HasPrefix(key, prefix) {
			values = append(values, cache)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *RedisService) deleteCache(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := cacheKey(subscriptionID, resourceGroup, name)
	if _, ok := s.caches[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Redis cache %q could not be found.", name))
	}
	delete(s.caches, key)
	delete(s.keys, key)
	delete(s.keyCounters, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *RedisService) listKeys(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	key := cacheKey(subscriptionID, resourceGroup, name)
	keys, ok := s.keys[key]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Redis cache %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, keys)
}

func (s *RedisService) regenerateKey(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		KeyType string `json:"keyType"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	keyName := strings.ToLower(input.KeyType)
	if keyName == "" {
		keyName = "primary"
	}
	if keyName != "primary" && keyName != "secondary" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidKeyType", "The keyType must be Primary or Secondary.")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := cacheKey(subscriptionID, resourceGroup, name)
	keys, ok := s.keys[key]
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Redis cache %q could not be found.", name))
	}
	if s.keyCounters[key] == nil {
		s.keyCounters[key] = map[string]int{"primary": 0, "secondary": 0}
	}
	s.keyCounters[key][keyName]++
	rotated := fmt.Sprintf("cloudmock-%s-%s-r%d", name, keyName, s.keyCounters[key][keyName])
	if keyName == "primary" {
		keys.PrimaryKey = rotated
	} else {
		keys.SecondaryKey = rotated
	}
	s.keys[key] = keys
	return azurearm.JSONResponse(http.StatusOK, keys)
}

type redisRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	CacheName      string
	Operation      string
}

func parseRoute(escapedPath string) (redisRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Cache") {
		return redisRoute{}, false
	}
	route := redisRoute{SubscriptionID: parts[1], ResourceGroup: parts[3], ResourceType: parts[6]}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.CacheName = parts[7]
		return route, true
	case 9:
		route.CacheName = parts[7]
		route.Operation = parts[8]
		return route, true
	default:
		return redisRoute{}, false
	}
}

func cacheID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Cache/Redis/" + name
}

func cacheKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func initialKeys(name string) accessKeys {
	return accessKeys{
		PrimaryKey:   "cloudmock-" + name + "-primary",
		SecondaryKey: "cloudmock-" + name + "-secondary",
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
