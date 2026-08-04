package appconfiguration

import (
	"encoding/base64"
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

const (
	controlPlaneAPIVersion = "2024-06-01"
	dataPlaneAPIVersion    = "2024-09-01"

	kvContentType          = "application/vnd.microsoft.appconfig.kv+json;charset=utf-8"
	kvSetContentType       = "application/vnd.microsoft.appconfig.kvset+json;charset=utf-8"
	keySetContentType      = "application/vnd.microsoft.appconfig.keyset+json;charset=utf-8"
	labelSetContentType    = "application/vnd.microsoft.appconfig.labelset+json;charset=utf-8"
	snapshotContentType    = "application/vnd.microsoft.appconfig.snapshot+json;charset=utf-8"
	snapshotSetContentType = "application/vnd.microsoft.appconfig.snapshotset+json;charset=utf-8"
	problemJSONType        = "application/problem+json;charset=utf-8"

	appConfigPageSize       = 100
	appConfigMaxCSVFilters  = 5
	snapshotListEscapedStar = '\x00'
)

type appConfigNameListItem struct {
	Token string
	Name  any
}

// AppConfigurationService implements Azure App Configuration control-plane
// stores plus first-slice SDK-facing key-value data-plane APIs.
type AppConfigurationService struct {
	mu          sync.RWMutex
	stores      map[string]ConfigurationStore
	keyValues   map[string]map[string]keyValue
	revisions   map[string][]keyValue
	snapshots   map[string]map[string]snapshot
	storeSeq    map[string]uint64
	storeSyncSN map[string]uint64
}

func New() *AppConfigurationService {
	return &AppConfigurationService{
		stores:      make(map[string]ConfigurationStore),
		keyValues:   make(map[string]map[string]keyValue),
		revisions:   make(map[string][]keyValue),
		snapshots:   make(map[string]map[string]snapshot),
		storeSeq:    make(map[string]uint64),
		storeSyncSN: make(map[string]uint64),
	}
}

func (s *AppConfigurationService) Name() string { return "Microsoft.AppConfiguration" }

func (s *AppConfigurationService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.AppConfiguration/configurationStores/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/configurationStores/read"},
		{Name: "ListByResourceGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/configurationStores/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.AppConfiguration/configurationStores/delete"},
		{Name: "ListStoreKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.AppConfiguration/configurationStores/listKeys/action"},
		{Name: "SetKeyValue", Method: http.MethodPut, IAMAction: "azure:Microsoft.AppConfiguration/kv/write"},
		{Name: "GetKeyValue", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/kv/read"},
		{Name: "ListKeyValues", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/kv/read"},
		{Name: "DeleteKeyValue", Method: http.MethodDelete, IAMAction: "azure:Microsoft.AppConfiguration/kv/delete"},
		{Name: "ListKeys", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/kv/read"},
		{Name: "ListLabels", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/kv/read"},
		{Name: "ListRevisions", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/kv/read"},
		{Name: "LockKeyValue", Method: http.MethodPut, IAMAction: "azure:Microsoft.AppConfiguration/kv/write"},
		{Name: "UnlockKeyValue", Method: http.MethodDelete, IAMAction: "azure:Microsoft.AppConfiguration/kv/write"},
		{Name: "CreateSnapshot", Method: http.MethodPut, IAMAction: "azure:Microsoft.AppConfiguration/snapshots/write"},
		{Name: "GetSnapshot", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/snapshots/read"},
		{Name: "ListSnapshots", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/snapshots/read"},
		{Name: "UpdateSnapshot", Method: http.MethodPatch, IAMAction: "azure:Microsoft.AppConfiguration/snapshots/archive/action"},
		{Name: "GetSnapshotOperation", Method: http.MethodGet, IAMAction: "azure:Microsoft.AppConfiguration/operations/read"},
	}
}

func (s *AppConfigurationService) HealthCheck() error { return nil }

func (s *AppConfigurationService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.AppConfiguration/configurationStores", APIVersion: controlPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.AppConfiguration/kv", APIVersion: dataPlaneAPIVersion},
	}
}

func (s *AppConfigurationService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.AppConfiguration/configurationStores")
}

func (s *AppConfigurationService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported App Configuration template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("App Configuration template resource is missing name")
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
	resp, err := s.createOrUpdateStore(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *AppConfigurationService) TemplateListOperationResult(subscriptionID, resourceGroup string, resource map[string]any, operation string) (any, bool) {
	if !strings.EqualFold(operation, "listKeys") || !s.SupportsTemplateResource(resource) {
		return nil, false
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, false
	}

	s.mu.RLock()
	store, ok := s.stores[storeKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return nil, false
	}
	return appConfigAccessKeys(store), true
}

func (s *AppConfigurationService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	var (
		resp *service.Response
		err  error
	)
	if storeName, parts, ok := dataPlaneStoreAndPath(ctx.RawRequest); ok {
		resp, err = s.handleDataPlane(ctx, storeName, parts)
	} else {
		resp, err = s.handleControlPlane(ctx)
	}
	applyAppConfigCommonResponseHeaders(ctx.RawRequest, resp)
	return resp, err
}

func (s *AppConfigurationService) handleControlPlane(ctx *service.RequestContext) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	providerIndex := segmentIndex(parts, "providers")
	if providerIndex < 4 || providerIndex+2 >= len(parts) ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[providerIndex+1], "Microsoft.AppConfiguration") ||
		!strings.EqualFold(parts[providerIndex+2], "configurationStores") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Configuration route is not implemented.")
	}

	subscriptionID := parts[1]
	resourceGroup := parts[3]
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listStores(subscriptionID, resourceGroup)
	}
	if len(parts) == providerIndex+5 && strings.EqualFold(parts[providerIndex+4], "listKeys") {
		if ctx.RawRequest.Method != http.MethodPost {
			return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
		}
		return s.listStoreKeys(subscriptionID, resourceGroup, parts[providerIndex+3])
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The App Configuration route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateStore(subscriptionID, resourceGroup, name, ctx.Body)
	case http.MethodGet:
		return s.getStore(subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		return s.deleteStore(subscriptionID, resourceGroup, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *AppConfigurationService) createOrUpdateStore(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
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
	if input.SKU == nil {
		input.SKU = map[string]any{"name": "Standard"}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	setDefault(input.Properties, "provisioningState", "Succeeded")
	setDefault(input.Properties, "creationDate", "2026-06-16T00:00:00+00:00")
	setDefault(input.Properties, "endpoint", "https://"+name+".azconfig.io")
	setDefault(input.Properties, "disableLocalAuth", false)
	setDefault(input.Properties, "privateEndpointConnections", []any{})
	setDefault(input.Properties, "dataPlaneProxy", map[string]any{"authenticationMode": "Local", "privateLinkDelegation": "Disabled"})
	setDefault(input.Properties, "softDeleteRetentionInDays", float64(30))
	setDefault(input.Properties, "defaultKeyValueRevisionRetentionPeriodInSeconds", float64(2592000))
	setDefault(input.Properties, "enablePurgeProtection", false)

	store := ConfigurationStore{
		ID:         storeID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.AppConfiguration/configurationStores",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Identity:   input.Identity,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := storeKey(subscriptionID, resourceGroup, name)
	_, existed := s.stores[key]
	s.stores[key] = store
	if s.keyValues[strings.ToLower(name)] == nil {
		s.keyValues[strings.ToLower(name)] = make(map[string]keyValue)
	}
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, store)
}

func (s *AppConfigurationService) getStore(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	store, ok := s.stores[storeKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Configuration store %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, store)
}

func (s *AppConfigurationService) listStores(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]ConfigurationStore, 0)
	for key, store := range s.stores {
		if strings.HasPrefix(key, prefix) {
			values = append(values, store)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *AppConfigurationService) deleteStore(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := storeKey(subscriptionID, resourceGroup, name)
	if _, ok := s.stores[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Configuration store %q could not be found.", name))
	}
	delete(s.stores, key)
	delete(s.keyValues, strings.ToLower(name))
	delete(s.revisions, strings.ToLower(name))
	delete(s.snapshots, strings.ToLower(name))
	return azurearm.JSONResponse(http.StatusOK, map[string]string{"status": "Succeeded"})
}

func (s *AppConfigurationService) listStoreKeys(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	store, ok := s.stores[storeKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Configuration store %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, appConfigAccessKeys(store))
}

func appConfigAccessKeys(store ConfigurationStore) map[string]any {
	endpoint := stringValue(store.Properties["endpoint"])
	if endpoint == "" {
		endpoint = "https://" + store.Name + ".azconfig.io"
	}

	keys := []struct {
		Name     string
		Suffix   string
		ReadOnly bool
	}{
		{Name: "Primary", Suffix: "primary"},
		{Name: "Secondary", Suffix: "secondary"},
		{Name: "Primary Read Only", Suffix: "primary-readonly", ReadOnly: true},
		{Name: "Secondary Read Only", Suffix: "secondary-readonly", ReadOnly: true},
	}
	values := make([]any, 0, len(keys))
	for _, key := range keys {
		id := strings.ToLower(store.Name) + "-" + key.Suffix
		secret := "cm-appconfig-" + strings.ToLower(store.Name) + "-" + key.Suffix
		values = append(values, map[string]any{
			"id":               id,
			"name":             key.Name,
			"value":            secret,
			"connectionString": "Endpoint=" + endpoint + ";Id=" + id + ";Secret=" + secret,
			"lastModified":     "2026-06-16T00:00:00+00:00",
			"readOnly":         key.ReadOnly,
		})
	}
	return map[string]any{"value": values}
}

func (s *AppConfigurationService) handleDataPlane(ctx *service.RequestContext, storeName string, parts []string) (*service.Response, error) {
	if resp, err := validateAppConfigDataPlaneAPIVersion(ctx.RawRequest); resp != nil || err != nil {
		return resp, err
	}

	if len(parts) == 0 || !strings.EqualFold(parts[0], "kv") {
		switch {
		case len(parts) == 1 && strings.EqualFold(parts[0], "keys"):
			if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
			}
			return s.listKeys(storeName, ctx.RawRequest)
		case len(parts) == 1 && strings.EqualFold(parts[0], "labels"):
			if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
			}
			return s.listLabels(storeName, ctx.RawRequest)
		case len(parts) == 1 && strings.EqualFold(parts[0], "revisions"):
			if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
			}
			return s.listRevisions(storeName, ctx.RawRequest)
		case len(parts) == 1 && strings.EqualFold(parts[0], "snapshots"):
			if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
			}
			return s.listSnapshots(storeName, ctx.RawRequest)
		case len(parts) == 1 && strings.EqualFold(parts[0], "operations"):
			if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
			}
			return s.getSnapshotOperation(storeName, ctx.RawRequest)
		case len(parts) == 2 && strings.EqualFold(parts[0], "snapshots"):
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.createSnapshot(storeName, parts[1], ctx.Body)
			case http.MethodGet, http.MethodHead:
				return s.getSnapshot(storeName, parts[1], ctx.RawRequest)
			case http.MethodPatch:
				return s.updateSnapshot(storeName, parts[1], ctx.RawRequest, ctx.Body)
			default:
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", parts[1], "")
			}
		case len(parts) == 2 && strings.EqualFold(parts[0], "locks"):
			switch ctx.RawRequest.Method {
			case http.MethodPut:
				return s.setLock(storeName, parts[1], ctx.RawRequest, true)
			case http.MethodDelete:
				return s.setLock(storeName, parts[1], ctx.RawRequest, false)
			default:
				return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", parts[1], "")
			}
		default:
			return appConfigProblemResponse(http.StatusNotImplemented, "https://azconfig.io/errors/not-implemented", "Route not implemented.", "", "")
		}
	}

	if len(parts) == 1 {
		if ctx.RawRequest.Method != http.MethodGet && ctx.RawRequest.Method != http.MethodHead {
			return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", "", "")
		}
		return s.listKeyValues(storeName, ctx.RawRequest)
	}
	if len(parts) != 2 {
		return appConfigProblemResponse(http.StatusNotImplemented, "https://azconfig.io/errors/not-implemented", "Route not implemented.", "", "")
	}

	key := parts[1]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.setKeyValue(storeName, key, ctx.RawRequest, ctx.Body)
	case http.MethodGet, http.MethodHead:
		return s.getKeyValue(storeName, key, ctx.RawRequest)
	case http.MethodDelete:
		return s.deleteKeyValue(storeName, key, ctx.RawRequest)
	default:
		return appConfigProblemResponse(http.StatusMethodNotAllowed, "https://azconfig.io/errors/method-not-allowed", "Method not allowed.", key, "")
	}
}

func (s *AppConfigurationService) setKeyValue(storeName, key string, req *http.Request, body []byte) (*service.Response, error) {
	label := normalizeLabel(req.URL.Query().Get("label"))
	store := strings.ToLower(storeName)
	kvKey := keyValueKey(key, label)

	var input struct {
		Value       any            `json:"value"`
		ContentType any            `json:"content_type"`
		Tags        map[string]any `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "The request content was invalid.", key, "Invalid JSON.")
		}
	}
	if input.Tags == nil {
		input.Tags = make(map[string]any)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.keyValues[store][kvKey]
	if !preconditionMatches(req.Header.Get("If-Match"), existing.ETag, exists) || preconditionNoneMatchFails(req.Header.Get("If-None-Match"), existing.ETag, exists) {
		return s.preconditionFailedLocked(store)
	}
	if exists && existing.Locked {
		return appConfigKeyLockedResponse(key)
	}
	if s.keyValues[store] == nil {
		s.keyValues[store] = make(map[string]keyValue)
	}
	s.storeSeq[store]++
	etag := fmt.Sprintf("%016x", s.storeSeq[store])
	kv := keyValue{
		Key:          key,
		Label:        label,
		Value:        input.Value,
		ContentType:  input.ContentType,
		Tags:         input.Tags,
		ETag:         etag,
		LastModified: appConfigRevisionTimestamp(s.storeSeq[store]),
		Locked:       false,
	}
	s.keyValues[store][kvKey] = kv
	s.revisions[store] = append(s.revisions[store], kv)

	return appConfigJSONResponse(http.StatusOK, keyValueResponseBody(kv, nil), kvContentType, map[string]string{
		"ETag":          quoteETag(etag),
		"Last-Modified": appConfigHTTPDate(kv.LastModified),
		"Sync-Token":    s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) getKeyValue(storeName, key string, req *http.Request) (*service.Response, error) {
	label := normalizeLabel(req.URL.Query().Get("label"))
	store := strings.ToLower(storeName)

	s.mu.Lock()
	defer s.mu.Unlock()

	kv, ok := s.keyValues[store][keyValueKey(key, label)]
	if !ok {
		return appConfigNotFoundResponse(key, s.nextSyncTokenLocked(store))
	}
	if !preconditionMatches(req.Header.Get("If-Match"), kv.ETag, true) {
		return s.preconditionFailedLocked(store)
	}
	if preconditionNoneMatchFails(req.Header.Get("If-None-Match"), kv.ETag, true) {
		return &service.Response{
			StatusCode: http.StatusNotModified,
			Headers: map[string]string{
				"ETag":       quoteETag(kv.ETag),
				"Sync-Token": s.nextSyncTokenLocked(store),
			},
		}, nil
	}

	return appConfigJSONResponse(http.StatusOK, keyValueResponseBody(kv, parseSelect(req.URL.Query())), kvContentType, map[string]string{
		"ETag":          quoteETag(kv.ETag),
		"Last-Modified": appConfigHTTPDate(kv.LastModified),
		"Sync-Token":    s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) listKeyValues(storeName string, req *http.Request) (*service.Response, error) {
	if resp, err := validateKeyValueListFilters(req.URL.Query()); resp != nil || err != nil {
		return resp, err
	}
	if snapshotName := strings.TrimSpace(req.URL.Query().Get("snapshot")); snapshotName != "" {
		return s.listSnapshotKeyValues(storeName, snapshotName, req)
	}

	store := strings.ToLower(storeName)
	keyFilter := req.URL.Query().Get("key")
	if keyFilter == "" {
		keyFilter = "*"
	}
	labelFilter, hasLabelFilter := req.URL.Query()["label"]
	tagFilters := req.URL.Query()["tags"]
	selectFields := parseSelect(req.URL.Query())
	acceptDatetime, mementoDatetime, historical, resp, err := appConfigAcceptDatetime(req)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	matching := make([]keyValue, 0)
	source := make([]keyValue, 0, len(s.keyValues[store]))
	if historical {
		source = appConfigHistoricalKeyValues(s.revisions[store], acceptDatetime)
	} else {
		for _, kv := range s.keyValues[store] {
			source = append(source, kv)
		}
	}
	for _, kv := range source {
		if !matchesWildcard(kv.Key, keyFilter) {
			continue
		}
		if hasLabelFilter && !matchesLabel(kv.Label, labelFilter) {
			continue
		}
		if !tagsMatch(kv.Tags, tagFilters) {
			continue
		}
		matching = append(matching, kv)
	}
	sortKeyValues(matching)
	listETag := appConfigListETag(matching)
	if !preconditionMatches(req.Header.Get("If-Match"), listETag, true) {
		return s.preconditionFailedLocked(store)
	}
	if preconditionNoneMatchFails(req.Header.Get("If-None-Match"), listETag, true) {
		return &service.Response{
			StatusCode: http.StatusNotModified,
			Headers: map[string]string{
				"ETag":       quoteETag(listETag),
				"Sync-Token": s.nextSyncTokenLocked(store),
			},
		}, nil
	}
	page, nextLink := paginateKeyValues(req, matching)

	items := make([]map[string]any, 0, len(page))
	for _, kv := range page {
		items = append(items, keyValueResponseBody(kv, selectFields))
	}
	body := map[string]any{"items": items}
	headers := map[string]string{
		"ETag":       quoteETag(listETag),
		"Sync-Token": s.nextSyncTokenLocked(store),
	}
	if nextLink != "" {
		body["@nextLink"] = nextLink
		headers["Link"] = "<" + nextLink + ">; rel=\"next\""
	}
	if historical {
		headers["Memento-Datetime"] = mementoDatetime
		originalLink := "<" + appConfigOriginalLink(req) + ">; rel=\"original\""
		if headers["Link"] != "" {
			headers["Link"] += ", " + originalLink
		} else {
			headers["Link"] = originalLink
		}
	}
	return appConfigJSONResponse(http.StatusOK, body, kvSetContentType, headers)
}

func (s *AppConfigurationService) listSnapshotKeyValues(storeName, snapshotName string, req *http.Request) (*service.Response, error) {
	store := strings.ToLower(storeName)
	keyFilter := req.URL.Query().Get("key")
	if keyFilter == "" {
		keyFilter = "*"
	}
	labelFilter, hasLabelFilter := req.URL.Query()["label"]
	tagFilters := req.URL.Query()["tags"]
	selectFields := parseSelect(req.URL.Query())

	s.mu.Lock()
	defer s.mu.Unlock()

	snap, ok := s.snapshots[store][snapshotName]
	if !ok {
		return appConfigNotFoundResponse(snapshotName, s.nextSyncTokenLocked(store))
	}

	items := make([]map[string]any, 0, len(snap.Items))
	for _, kv := range snap.Items {
		if !matchesWildcard(kv.Key, keyFilter) {
			continue
		}
		if hasLabelFilter && !matchesLabel(kv.Label, labelFilter) {
			continue
		}
		if !tagsMatch(kv.Tags, tagFilters) {
			continue
		}
		items = append(items, keyValueResponseBody(kv, selectFields))
	}
	sort.Slice(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["key"])+fmt.Sprint(items[i]["label"]) < fmt.Sprint(items[j]["key"])+fmt.Sprint(items[j]["label"])
	})

	return appConfigJSONResponse(http.StatusOK, map[string]any{"items": items}, kvSetContentType, map[string]string{
		"Sync-Token": s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) listKeys(storeName string, req *http.Request) (*service.Response, error) {
	store := strings.ToLower(storeName)
	nameFilters := appConfigNameFilterValues(req.URL.Query()["name"])
	if len(nameFilters) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("name")
	}
	if len(nameFilters) == 0 {
		nameFilters = []string{"*"}
	}
	selectFields := parseSelect(req.URL.Query())
	acceptDatetime, mementoDatetime, historical, resp, err := appConfigAcceptDatetime(req)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool)
	names := make([]appConfigNameListItem, 0)
	for _, kv := range s.appConfigKeyValueNameSourceLocked(store, historical, acceptDatetime) {
		if seen[kv.Key] || !matchesAppConfigNameFilters(kv.Key, nameFilters) {
			continue
		}
		seen[kv.Key] = true
		names = append(names, appConfigNameListItem{Token: kv.Key, Name: kv.Key})
	}
	return s.appConfigNameListResponseLocked(store, req, names, selectFields, keySetContentType, historical, mementoDatetime)
}

func (s *AppConfigurationService) listLabels(storeName string, req *http.Request) (*service.Response, error) {
	store := strings.ToLower(storeName)
	nameFilters := appConfigNameFilterValues(req.URL.Query()["name"])
	if len(nameFilters) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("name")
	}
	if len(nameFilters) == 0 {
		nameFilters = []string{"*"}
	}
	selectFields := parseSelect(req.URL.Query())
	acceptDatetime, mementoDatetime, historical, resp, err := appConfigAcceptDatetime(req)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]bool)
	names := make([]appConfigNameListItem, 0)
	for _, kv := range s.appConfigKeyValueNameSourceLocked(store, historical, acceptDatetime) {
		filterValue := kv.Label
		if filterValue == "" {
			filterValue = "\x00"
		}
		if seen[kv.Label] || !matchesAppConfigNameFilters(filterValue, nameFilters) {
			continue
		}
		seen[kv.Label] = true
		name := any(nil)
		if kv.Label != "" {
			name = kv.Label
		}
		names = append(names, appConfigNameListItem{Token: kv.Label, Name: name})
	}
	return s.appConfigNameListResponseLocked(store, req, names, selectFields, labelSetContentType, historical, mementoDatetime)
}

func (s *AppConfigurationService) appConfigKeyValueNameSourceLocked(store string, historical bool, at time.Time) []keyValue {
	if historical {
		return appConfigHistoricalKeyValues(s.revisions[store], at)
	}
	source := make([]keyValue, 0, len(s.keyValues[store]))
	for _, kv := range s.keyValues[store] {
		source = append(source, kv)
	}
	return source
}

func (s *AppConfigurationService) appConfigNameListResponseLocked(store string, req *http.Request, names []appConfigNameListItem, selectFields []string, contentType string, historical bool, mementoDatetime string) (*service.Response, error) {
	sort.Slice(names, func(i, j int) bool { return names[i].Token < names[j].Token })
	page, nextLink := paginateAppConfigNameItems(req, names)

	items := make([]map[string]any, 0, len(page))
	for _, item := range page {
		items = append(items, projectAppConfigFields(map[string]any{"name": item.Name}, selectFields))
	}
	body := map[string]any{"items": items}
	headers := map[string]string{
		"Sync-Token": s.nextSyncTokenLocked(store),
	}
	if nextLink != "" {
		body["@nextLink"] = nextLink
		headers["Link"] = "<" + nextLink + ">; rel=\"next\""
	}
	if historical {
		headers["Memento-Datetime"] = mementoDatetime
		originalLink := "<" + appConfigOriginalLink(req) + ">; rel=\"original\""
		if headers["Link"] != "" {
			headers["Link"] += ", " + originalLink
		} else {
			headers["Link"] = originalLink
		}
	}
	return appConfigJSONResponse(http.StatusOK, body, contentType, headers)
}

func (s *AppConfigurationService) listRevisions(storeName string, req *http.Request) (*service.Response, error) {
	if resp, err := validateRevisionListFilters(req.URL.Query()); resp != nil || err != nil {
		return resp, err
	}

	store := strings.ToLower(storeName)
	keyFilter := req.URL.Query().Get("key")
	if keyFilter == "" {
		keyFilter = "*"
	}
	labelFilter, hasLabelFilter := req.URL.Query()["label"]
	tagFilters := req.URL.Query()["tags"]
	selectFields := parseSelect(req.URL.Query())
	acceptDatetime, mementoDatetime, historical, resp, err := appConfigAcceptDatetime(req)
	if resp != nil || err != nil {
		return resp, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items := make([]map[string]any, 0)
	for _, kv := range s.revisions[store] {
		if historical {
			modified, ok := parseAppConfigDatetime(kv.LastModified)
			if !ok || modified.After(acceptDatetime) {
				continue
			}
		}
		if !matchesRevisionWildcard(kv.Key, keyFilter) {
			continue
		}
		if hasLabelFilter && !matchesRevisionLabel(kv.Label, labelFilter) {
			continue
		}
		if !tagsMatch(kv.Tags, tagFilters) {
			continue
		}
		items = append(items, keyValueResponseBody(kv, selectFields))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return fmt.Sprint(items[i]["last_modified"])+fmt.Sprint(items[i]["etag"]) > fmt.Sprint(items[j]["last_modified"])+fmt.Sprint(items[j]["etag"])
	})
	total := len(items)
	statusCode := http.StatusOK
	headers := map[string]string{
		"Accept-Ranges": "items",
		"Sync-Token":    s.nextSyncTokenLocked(store),
	}
	if historical {
		headers["Memento-Datetime"] = mementoDatetime
		headers["Link"] = "<" + appConfigOriginalLink(req) + ">; rel=\"original\""
	}
	if req.Header.Get("Range") != "" {
		start, end, ok := parseAppConfigItemRange(req.Header.Get("Range"))
		if !ok || start >= total {
			headers["Content-Range"] = fmt.Sprintf("items */%d", total)
			return appConfigProblemResponseWithHeaders(http.StatusRequestedRangeNotSatisfiable, "https://azconfig.io/errors/range-not-satisfiable", "The requested range could not be satisfied.", "Range", "", headers)
		}
		if end >= total {
			end = total - 1
		}
		items = items[start : end+1]
		statusCode = http.StatusPartialContent
		headers["Content-Range"] = fmt.Sprintf("items %d-%d/%d", start, end, total)
	} else {
		var nextLink string
		items, nextLink = paginateRevisionItems(req, items)
		if nextLink != "" {
			nextHeader := "<" + nextLink + ">; rel=\"next\""
			if headers["Link"] != "" {
				headers["Link"] = nextHeader + ", " + headers["Link"]
			} else {
				headers["Link"] = nextHeader
			}
		}
		body := map[string]any{"items": items}
		if nextLink != "" {
			body["@nextLink"] = nextLink
		}
		return appConfigJSONResponse(statusCode, body, kvSetContentType, headers)
	}
	return appConfigJSONResponse(statusCode, map[string]any{"items": items}, kvSetContentType, headers)
}

func (s *AppConfigurationService) deleteKeyValue(storeName, key string, req *http.Request) (*service.Response, error) {
	label := normalizeLabel(req.URL.Query().Get("label"))
	store := strings.ToLower(storeName)
	kvKey := keyValueKey(key, label)

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.keyValues[store][kvKey]
	if !preconditionMatches(req.Header.Get("If-Match"), existing.ETag, exists) || preconditionNoneMatchFails(req.Header.Get("If-None-Match"), existing.ETag, exists) {
		return s.preconditionFailedLocked(store)
	}
	if !exists {
		return &service.Response{
			StatusCode: http.StatusNoContent,
			Headers: map[string]string{
				"Sync-Token": s.nextSyncTokenLocked(store),
			},
		}, nil
	}
	delete(s.keyValues[store], kvKey)

	return appConfigJSONResponse(http.StatusOK, keyValueResponseBody(existing, nil), kvContentType, map[string]string{
		"ETag":          quoteETag(existing.ETag),
		"Last-Modified": appConfigHTTPDate(existing.LastModified),
		"Sync-Token":    s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) setLock(storeName, key string, req *http.Request, locked bool) (*service.Response, error) {
	label := normalizeLabel(req.URL.Query().Get("label"))
	store := strings.ToLower(storeName)
	kvKey := keyValueKey(key, label)

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.keyValues[store][kvKey]
	if !preconditionMatches(req.Header.Get("If-Match"), existing.ETag, exists) || preconditionNoneMatchFails(req.Header.Get("If-None-Match"), existing.ETag, exists) {
		return s.preconditionFailedLocked(store)
	}
	if !exists {
		return appConfigNotFoundResponse(key, s.nextSyncTokenLocked(store))
	}
	s.storeSeq[store]++
	existing.ETag = fmt.Sprintf("%016x", s.storeSeq[store])
	existing.LastModified = appConfigRevisionTimestamp(s.storeSeq[store])
	existing.Locked = locked
	s.keyValues[store][kvKey] = existing
	s.revisions[store] = append(s.revisions[store], existing)

	return appConfigJSONResponse(http.StatusOK, keyValueResponseBody(existing, nil), kvContentType, map[string]string{
		"ETag":          quoteETag(existing.ETag),
		"Last-Modified": appConfigHTTPDate(existing.LastModified),
		"Sync-Token":    s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) createSnapshot(storeName, name string, body []byte) (*service.Response, error) {
	store := strings.ToLower(storeName)
	var input struct {
		Filters         []snapshotFilter  `json:"filters"`
		CompositionType string            `json:"composition_type"`
		RetentionPeriod float64           `json:"retention_period"`
		Tags            map[string]string `json:"tags"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "The request content was invalid.", name, "Invalid JSON.")
		}
	}
	if input.CompositionType == "" {
		input.CompositionType = "key"
	}
	input.CompositionType = strings.ToLower(input.CompositionType)
	if input.CompositionType != "key" && input.CompositionType != "key_label" {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter 'composition_type'.", "composition_type", "The composition_type must be 'key' or 'key_label'.")
	}
	if len(input.Filters) == 0 {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter 'filters'.", "filters", "At least one snapshot filter is required.")
	}
	if len(input.Filters) > 3 {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter 'filters'.", "filters", "At most three snapshot filters are allowed.")
	}
	for i := range input.Filters {
		filterName := fmt.Sprintf("filters[%d]", i)
		input.Filters[i].Key = strings.TrimSpace(input.Filters[i].Key)
		if input.Filters[i].Key == "" {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter '"+filterName+".key'.", filterName+".key", "A snapshot filter key is required.")
		}
		input.Filters[i].Label = normalizeLabel(input.Filters[i].Label)
		if input.CompositionType == "key" && isMultiMatchSnapshotLabel(input.Filters[i].Label) {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter '"+filterName+".label'.", filterName+".label", "Multi-match label filters are not supported with key composition snapshots.")
		}
		if len(input.Filters[i].Tags) > 5 {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter '"+filterName+".tags'.", filterName+".tags", "At most five tag filters are allowed.")
		}
	}
	if input.RetentionPeriod == 0 {
		input.RetentionPeriod = 2592000
	}
	if input.Tags == nil {
		input.Tags = make(map[string]string)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.snapshots[store] == nil {
		s.snapshots[store] = make(map[string]snapshot)
	}
	if _, exists := s.snapshots[store][name]; exists {
		return appConfigProblemResponse(http.StatusConflict, "https://azconfig.io/errors/already-exists", "The snapshot already exists.", name, "")
	}

	items := s.captureSnapshotItemsLocked(store, input.Filters, input.CompositionType)
	s.storeSeq[store]++
	etag := fmt.Sprintf("%016x", s.storeSeq[store])
	snap := snapshot{
		Name:             name,
		Status:           "ready",
		Filters:          append([]snapshotFilter(nil), input.Filters...),
		CompositionType:  input.CompositionType,
		Created:          "2026-06-16T00:00:00.0000000Z",
		Expires:          nil,
		RetentionSeconds: input.RetentionPeriod,
		ItemsCount:       len(items),
		Size:             snapshotItemSize(items),
		Tags:             cloneStringMap(input.Tags),
		ETag:             etag,
		Items:            items,
	}
	s.snapshots[store][name] = snap

	createBody := snap
	createBody.Status = "provisioning"
	createBody.ItemsCount = 0
	createBody.Size = 0
	createBody.Items = nil

	return appConfigJSONResponse(http.StatusCreated, snapshotResponseBody(createBody, nil), snapshotContentType, map[string]string{
		"ETag":               quoteETag(etag),
		"Operation-Location": "/operations?snapshot=" + url.QueryEscape(name) + "&api-version=" + dataPlaneAPIVersion,
		"Sync-Token":         s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) getSnapshotOperation(storeName string, req *http.Request) (*service.Response, error) {
	name := strings.TrimSpace(req.URL.Query().Get("snapshot"))
	if name == "" {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter 'snapshot'.", "snapshot", "A snapshot name is required.")
	}

	store := strings.ToLower(storeName)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.snapshots[store][name]; !ok {
		return appConfigNotFoundResponse(name, s.nextSyncTokenLocked(store))
	}
	return appConfigJSONResponse(http.StatusOK, map[string]any{
		"id":     name,
		"status": "Succeeded",
		"error":  nil,
	}, "application/json;charset=utf-8", map[string]string{
		"Sync-Token": s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) getSnapshot(storeName, name string, req *http.Request) (*service.Response, error) {
	store := strings.ToLower(storeName)

	s.mu.Lock()
	defer s.mu.Unlock()

	snap, ok := s.snapshots[store][name]
	if !ok {
		return appConfigNotFoundResponse(name, s.nextSyncTokenLocked(store))
	}
	if !preconditionMatches(req.Header.Get("If-Match"), snap.ETag, true) {
		return s.preconditionFailedLocked(store)
	}
	if preconditionNoneMatchFails(req.Header.Get("If-None-Match"), snap.ETag, true) {
		return &service.Response{
			StatusCode: http.StatusNotModified,
			Headers: map[string]string{
				"ETag":       quoteETag(snap.ETag),
				"Sync-Token": s.nextSyncTokenLocked(store),
			},
		}, nil
	}

	apiVersion := req.URL.Query().Get("api-version")
	if apiVersion == "" {
		apiVersion = dataPlaneAPIVersion
	}
	return appConfigJSONResponse(http.StatusOK, snapshotResponseBody(snap, parseSelect(req.URL.Query())), snapshotContentType, map[string]string{
		"ETag":       quoteETag(snap.ETag),
		"Link":       "</kv?snapshot=" + url.QueryEscape(name) + "&api-version=" + url.QueryEscape(apiVersion) + ">; rel=\"items\"",
		"Sync-Token": s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) listSnapshots(storeName string, req *http.Request) (*service.Response, error) {
	store := strings.ToLower(storeName)
	nameFilters := snapshotListFilterValues(req.URL.Query()["name"])
	if len(nameFilters) == 0 {
		nameFilters = []string{"*"}
	}
	if len(nameFilters) > appConfigMaxCSVFilters {
		return snapshotListFilterProblem("name")
	}
	statusFilters := snapshotListFilterValues(req.URL.Query()["status"])
	if len(statusFilters) > appConfigMaxCSVFilters {
		return snapshotListFilterProblem("status")
	}
	selectFields := parseSelect(req.URL.Query())

	s.mu.Lock()
	defer s.mu.Unlock()

	matching := make([]snapshot, 0)
	for _, snap := range s.snapshots[store] {
		if !matchesSnapshotName(snap.Name, nameFilters) {
			continue
		}
		if len(statusFilters) > 0 && !matchesSnapshotStatus(snap.Status, statusFilters) {
			continue
		}
		matching = append(matching, snap)
	}
	sort.Slice(matching, func(i, j int) bool { return matching[i].Name < matching[j].Name })
	page, nextLink := paginateSnapshots(req, matching)

	items := make([]map[string]any, 0, len(page))
	for _, snap := range page {
		items = append(items, snapshotResponseBody(snap, selectFields))
	}
	body := map[string]any{"items": items}
	headers := map[string]string{
		"Sync-Token": s.nextSyncTokenLocked(store),
	}
	if nextLink != "" {
		body["@nextLink"] = nextLink
		headers["Link"] = "<" + nextLink + ">; rel=\"next\""
	}
	return appConfigJSONResponse(http.StatusOK, body, snapshotSetContentType, headers)
}

func (s *AppConfigurationService) updateSnapshot(storeName, name string, req *http.Request, body []byte) (*service.Response, error) {
	store := strings.ToLower(storeName)
	var input struct {
		Status string `json:"status"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "The request content was invalid.", name, "Invalid JSON.")
		}
	}
	input.Status = strings.ToLower(strings.TrimSpace(input.Status))
	if input.Status != "archived" && input.Status != "ready" {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "The requested snapshot status is invalid.", name, "")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	snap, ok := s.snapshots[store][name]
	if !ok {
		return appConfigNotFoundResponse(name, s.nextSyncTokenLocked(store))
	}
	if !preconditionMatches(req.Header.Get("If-Match"), snap.ETag, true) || preconditionNoneMatchFails(req.Header.Get("If-None-Match"), snap.ETag, true) {
		return s.preconditionFailedLocked(store)
	}
	if input.Status == snap.Status {
		return appConfigJSONResponse(http.StatusOK, snapshotResponseBody(snap, nil), snapshotContentType, map[string]string{
			"ETag":       quoteETag(snap.ETag),
			"Sync-Token": s.nextSyncTokenLocked(store),
		})
	}
	if input.Status == "archived" && snap.Status != "ready" {
		return appConfigProblemResponse(http.StatusConflict, "https://azconfig.io/errors/invalid-state", "Target resource state invalid.", name, "The target resource is not in a valid state to perform the requested operation.")
	}
	if input.Status == "ready" && snap.Status != "archived" {
		return appConfigProblemResponse(http.StatusConflict, "https://azconfig.io/errors/invalid-state", "Target resource state invalid.", name, "The target resource is not in a valid state to perform the requested operation.")
	}

	s.storeSeq[store]++
	snap.ETag = fmt.Sprintf("%016x", s.storeSeq[store])
	snap.Status = input.Status
	if input.Status == "archived" {
		snap.Expires = "2026-06-23T00:00:00.0000000Z"
	} else {
		snap.Expires = nil
	}
	s.snapshots[store][name] = snap

	return appConfigJSONResponse(http.StatusOK, snapshotResponseBody(snap, nil), snapshotContentType, map[string]string{
		"ETag":       quoteETag(snap.ETag),
		"Sync-Token": s.nextSyncTokenLocked(store),
	})
}

func (s *AppConfigurationService) captureSnapshotItemsLocked(store string, filters []snapshotFilter, composition string) []keyValue {
	source := make([]keyValue, 0, len(s.keyValues[store]))
	for _, kv := range s.keyValues[store] {
		source = append(source, cloneKeyValue(kv))
	}
	sort.Slice(source, func(i, j int) bool {
		return source[i].LastModified+source[i].ETag < source[j].LastModified+source[j].ETag
	})

	if strings.EqualFold(composition, "key_label") {
		seen := make(map[string]bool)
		items := make([]keyValue, 0)
		for _, kv := range source {
			for _, filter := range filters {
				if snapshotFilterMatches(kv, filter) {
					id := keyValueKey(kv.Key, kv.Label)
					if !seen[id] {
						seen[id] = true
						items = append(items, cloneKeyValue(kv))
					}
					break
				}
			}
		}
		sortKeyValues(items)
		return items
	}

	byKey := make(map[string]keyValue)
	for _, filter := range filters {
		for _, kv := range source {
			if snapshotFilterMatches(kv, filter) {
				byKey[kv.Key] = cloneKeyValue(kv)
			}
		}
	}
	items := make([]keyValue, 0, len(byKey))
	for _, kv := range byKey {
		items = append(items, kv)
	}
	sortKeyValues(items)
	return items
}

func snapshotFilterMatches(kv keyValue, filter snapshotFilter) bool {
	keyFilter := filter.Key
	if keyFilter == "" {
		keyFilter = "*"
	}
	if !matchesWildcard(kv.Key, keyFilter) {
		return false
	}
	if !tagsMatch(kv.Tags, filter.Tags) {
		return false
	}
	if filter.Label == "" || filter.Label == "*" {
		return true
	}
	return matchesLabel(kv.Label, []string{filter.Label})
}

func isMultiMatchSnapshotLabel(label string) bool {
	return label == "*" || strings.Contains(label, ",")
}

func snapshotResponseBody(snap snapshot, selectFields []string) map[string]any {
	body := map[string]any{
		"name":             snap.Name,
		"status":           snap.Status,
		"filters":          snap.Filters,
		"composition_type": snap.CompositionType,
		"created":          snap.Created,
		"expires":          snap.Expires,
		"retention_period": snap.RetentionSeconds,
		"size":             snap.Size,
		"items_count":      snap.ItemsCount,
		"tags":             snap.Tags,
		"etag":             snap.ETag,
	}
	return projectAppConfigFields(body, selectFields)
}

func snapshotItemSize(items []keyValue) int {
	bodies := make([]map[string]any, 0, len(items))
	for _, kv := range items {
		bodies = append(bodies, keyValueResponseBody(kv, nil))
	}
	data, err := gojson.Marshal(bodies)
	if err != nil {
		return 0
	}
	return len(data)
}

func validateKeyValueListFilters(values url.Values) (*service.Response, error) {
	if len(snapshotListFilterValues(values["key"])) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("key")
	}
	if !validAppConfigNameFilters(values["key"]) {
		return appConfigInvalidFilterProblem("key")
	}
	if len(snapshotListFilterValues(values["label"])) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("label")
	}
	if !validAppConfigNameFilters(values["label"]) {
		return appConfigInvalidFilterProblem("label")
	}
	if len(values["tags"]) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("tags")
	}
	for _, filter := range values["tags"] {
		if !validAppConfigTagFilter(filter) {
			return appConfigInvalidFilterProblem("tags")
		}
	}
	return nil, nil
}

func validateRevisionListFilters(values url.Values) (*service.Response, error) {
	if len(snapshotListFilterValues(values["key"])) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("key")
	}
	if !validRevisionNameFilters(values["key"]) {
		return appConfigInvalidFilterProblem("key")
	}
	if len(snapshotListFilterValues(values["label"])) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("label")
	}
	if !validRevisionNameFilters(values["label"]) {
		return appConfigInvalidFilterProblem("label")
	}
	if len(values["tags"]) > appConfigMaxCSVFilters {
		return appConfigFilterProblem("tags")
	}
	for _, filter := range values["tags"] {
		if !validAppConfigTagFilter(filter) {
			return appConfigInvalidFilterProblem("tags")
		}
	}
	return nil, nil
}

func validAppConfigNameFilters(filters []string) bool {
	for _, raw := range filters {
		current := []rune{}
		escaped := false
		for _, r := range raw {
			if escaped {
				current = append(current, r)
				escaped = false
				continue
			}
			if r == '\\' {
				escaped = true
				continue
			}
			if r == ',' {
				if !validAppConfigNameFilterRunes(current) {
					return false
				}
				current = current[:0]
				continue
			}
			current = append(current, r)
		}
		if escaped {
			current = append(current, '\\')
		}
		if !validAppConfigNameFilterRunes(current) {
			return false
		}
	}
	return true
}

func validRevisionNameFilters(filters []string) bool {
	for _, filter := range snapshotListFilterValues(filters) {
		if !validRevisionNameFilter(filter) {
			return false
		}
	}
	return true
}

func validRevisionNameFilter(filter string) bool {
	runes := []rune(filter)
	for i, r := range runes {
		if r == snapshotListEscapedStar {
			continue
		}
		if r == '*' && i != 0 && i != len(runes)-1 {
			return false
		}
	}
	return true
}

func validAppConfigNameFilterRunes(value []rune) bool {
	for i, r := range value {
		if r == '*' && i != len(value)-1 {
			return false
		}
	}
	return true
}

func validAppConfigTagFilter(filter string) bool {
	if filter == "" {
		return true
	}
	_, value, ok := strings.Cut(filter, "=")
	if !ok || value == "" || value == "\x00" {
		return true
	}
	escaped := false
	for _, r := range value {
		if escaped {
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		switch r {
		case '*', ',':
			return false
		}
	}
	return !escaped
}

func paginateSnapshots(req *http.Request, snapshots []snapshot) ([]snapshot, string) {
	start := 0
	if decoded := decodeAppConfigContinuationToken(appConfigContinuationToken(req.URL.Query())); decoded != "" {
		for start < len(snapshots) && snapshots[start].Name <= decoded {
			start++
		}
	}
	end := start + appConfigPageSize
	if end > len(snapshots) {
		end = len(snapshots)
	}
	if start > len(snapshots) {
		start = len(snapshots)
	}
	page := snapshots[start:end]
	if end >= len(snapshots) {
		return page, ""
	}
	return page, buildAppConfigNextLink(req, encodeAppConfigContinuationToken(snapshots[end-1].Name))
}

func paginateKeyValues(req *http.Request, values []keyValue) ([]keyValue, string) {
	start := 0
	if decoded := decodeAppConfigContinuationToken(appConfigContinuationToken(req.URL.Query())); decoded != "" {
		for start < len(values) && keyValueSortToken(values[start]) <= decoded {
			start++
		}
	}
	end := start + appConfigPageSize
	if end > len(values) {
		end = len(values)
	}
	if start > len(values) {
		start = len(values)
	}
	page := values[start:end]
	if end >= len(values) {
		return page, ""
	}
	return page, buildAppConfigNextLink(req, encodeAppConfigContinuationToken(keyValueSortToken(values[end-1])))
}

func paginateRevisionItems(req *http.Request, items []map[string]any) ([]map[string]any, string) {
	start := 0
	if decoded := decodeAppConfigContinuationToken(appConfigContinuationToken(req.URL.Query())); decoded != "" {
		for start < len(items) && revisionSortToken(items[start]) >= decoded {
			start++
		}
	}
	end := start + appConfigPageSize
	if end > len(items) {
		end = len(items)
	}
	if start > len(items) {
		start = len(items)
	}
	page := items[start:end]
	if end >= len(items) {
		return page, ""
	}
	return page, buildAppConfigNextLink(req, encodeAppConfigContinuationToken(revisionSortToken(items[end-1])))
}

func paginateAppConfigNameItems(req *http.Request, items []appConfigNameListItem) ([]appConfigNameListItem, string) {
	start := 0
	if decoded := decodeAppConfigContinuationToken(appConfigContinuationToken(req.URL.Query())); decoded != "" {
		for start < len(items) && items[start].Token <= decoded {
			start++
		}
	}
	end := start + appConfigPageSize
	if end > len(items) {
		end = len(items)
	}
	if start > len(items) {
		start = len(items)
	}
	page := items[start:end]
	if end >= len(items) {
		return page, ""
	}
	return page, buildAppConfigNextLink(req, encodeAppConfigContinuationToken(items[end-1].Token))
}

func keyValueSortToken(kv keyValue) string {
	return kv.Key + "\x00" + kv.Label
}

func revisionSortToken(item map[string]any) string {
	return fmt.Sprint(item["last_modified"]) + "\x00" + fmt.Sprint(item["etag"])
}

func appConfigContinuationToken(values url.Values) string {
	for key, options := range values {
		if strings.EqualFold(key, "after") && len(options) > 0 {
			return options[0]
		}
	}
	return ""
}

func encodeAppConfigContinuationToken(value string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(value))
}

func decodeAppConfigContinuationToken(value string) string {
	if value == "" {
		return ""
	}
	decoded, err := base64.RawURLEncoding.DecodeString(value)
	if err != nil {
		return ""
	}
	return string(decoded)
}

func buildAppConfigNextLink(req *http.Request, token string) string {
	values := url.Values{}
	apiVersion := req.URL.Query().Get("api-version")
	if apiVersion == "" {
		apiVersion = dataPlaneAPIVersion
	}
	values.Set("api-version", apiVersion)
	for key, options := range req.URL.Query() {
		if strings.EqualFold(key, "api-version") || strings.EqualFold(key, "after") {
			continue
		}
		for _, option := range options {
			values.Add(key, option)
		}
	}
	values.Set("after", token)

	path := req.URL.EscapedPath()
	if path == "" {
		path = "/snapshots"
	}
	return path + "?" + values.Encode()
}

func matchesSnapshotStatus(status string, filters []string) bool {
	for _, filter := range filters {
		if filter == "*" || strings.EqualFold(filter, status) {
			return true
		}
	}
	return false
}

func matchesSnapshotName(name string, filters []string) bool {
	for _, filter := range filters {
		if matchesSnapshotNameFilter(name, filter) {
			return true
		}
	}
	return false
}

func matchesSnapshotNameFilter(name, filter string) bool {
	if filter == "" || filter == "*" {
		return true
	}
	if strings.HasSuffix(filter, "*") {
		return strings.HasPrefix(name, unescapeSnapshotListFilterValue(strings.TrimSuffix(filter, "*")))
	}
	return name == unescapeSnapshotListFilterValue(filter)
}

func snapshotListFilterValues(raw []string) []string {
	values := make([]string, 0, len(raw))
	for _, filter := range raw {
		values = append(values, splitSnapshotListFilter(filter)...)
	}
	return values
}

func appConfigNameFilterValues(raw []string) []string {
	return snapshotListFilterValues(raw)
}

func matchesAppConfigNameFilters(value string, filters []string) bool {
	for _, filter := range filters {
		if matchesAppConfigNameFilter(value, filter) {
			return true
		}
	}
	return false
}

func matchesAppConfigNameFilter(value, filter string) bool {
	if filter == "" || filter == "*" {
		return true
	}
	if strings.HasSuffix(filter, "*") {
		return strings.HasPrefix(value, unescapeSnapshotListFilterValue(strings.TrimSuffix(filter, "*")))
	}
	return value == unescapeSnapshotListFilterValue(filter)
}

func splitSnapshotListFilter(filter string) []string {
	values := make([]string, 0, 1)
	var current strings.Builder
	escaped := false
	flush := func() {
		option := strings.TrimSpace(current.String())
		if option != "" {
			values = append(values, option)
		}
		current.Reset()
	}

	for _, r := range filter {
		if escaped {
			if r == '*' {
				current.WriteRune(snapshotListEscapedStar)
			} else {
				current.WriteRune(r)
			}
			escaped = false
			continue
		}
		switch r {
		case '\\':
			escaped = true
		case ',':
			flush()
		default:
			current.WriteRune(r)
		}
	}
	if escaped {
		current.WriteRune('\\')
	}
	flush()
	return values
}

func unescapeSnapshotListFilterValue(filter string) string {
	return strings.ReplaceAll(filter, string(snapshotListEscapedStar), "*")
}

func snapshotListFilterProblem(field string) (*service.Response, error) {
	return appConfigFilterProblem(field)
}

func appConfigFilterProblem(field string) (*service.Response, error) {
	return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter '"+field+"'.", field, "At most five "+field+" filters are allowed.")
}

func appConfigInvalidFilterProblem(field string) (*service.Response, error) {
	return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request parameter '"+field+"'.", field, "Filter contains an unescaped reserved character.")
}

func appConfigKeyLockedResponse(key string) (*service.Response, error) {
	return appConfigProblemResponse(http.StatusConflict, "https://azconfig.io/errors/key-locked", "Modifying key '"+key+"' is not allowed", key, "The key is read-only. To allow modification unlock it first.")
}

func validateAppConfigDataPlaneAPIVersion(req *http.Request) (*service.Response, error) {
	versions := req.URL.Query()["api-version"]
	if len(versions) == 0 || strings.TrimSpace(versions[0]) == "" {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "API version is not specified", "api-version", "An API version is required, but was not specified.")
	}
	if len(versions) > 1 {
		requested := make([]string, 0, len(versions))
		for _, version := range versions {
			requested = append(requested, strings.TrimSpace(version))
		}
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Ambiguous API version", "api-version", "The following API versions were requested: "+strings.Join(requested, ", ")+". At most, only a single API version may be specified. Please update the intended API version and retry the request.")
	}
	version := strings.TrimSpace(versions[0])
	requestURI := appConfigRequestURIWithoutQuery(req)
	if !isAppConfigAPIVersionSyntax(version) {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid API version", "api-version", "The HTTP resource that matches the request URI '"+requestURI+"' does not support the API version '"+version+"'.")
	}
	if version != dataPlaneAPIVersion {
		return appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Unsupported API version", "api-version", "The HTTP resource that matches the request URI '"+requestURI+"' does not support the API version '"+version+"'.")
	}
	return nil, nil
}

func isAppConfigAPIVersionSyntax(version string) bool {
	if version == "" {
		return false
	}
	if parts := strings.Split(version, "."); len(parts) == 2 {
		for _, part := range parts {
			if part == "" {
				return false
			}
			for _, r := range part {
				if r < '0' || r > '9' {
					return false
				}
			}
		}
		return true
	}
	if len(version) != len("2006-01-02") {
		return false
	}
	if _, err := time.Parse("2006-01-02", version); err != nil {
		return false
	}
	return true
}

func appConfigRequestURIWithoutQuery(req *http.Request) string {
	if req.URL == nil {
		return ""
	}
	path := req.URL.EscapedPath()
	if path == "" {
		return "/"
	}
	return path
}

func applyAppConfigCommonResponseHeaders(req *http.Request, resp *service.Response) {
	if req == nil || resp == nil {
		return
	}
	if !strings.EqualFold(strings.TrimSpace(req.Header.Get("x-ms-return-client-request-id")), "true") {
		return
	}
	clientRequestID := strings.TrimSpace(req.Header.Get("x-ms-client-request-id"))
	if clientRequestID == "" {
		return
	}
	if resp.Headers == nil {
		resp.Headers = make(map[string]string)
	}
	resp.Headers["x-ms-client-request-id"] = clientRequestID
}

func sortKeyValues(values []keyValue) {
	sort.Slice(values, func(i, j int) bool {
		left := values[i].Key + "\x00" + values[i].Label
		right := values[j].Key + "\x00" + values[j].Label
		return left < right
	})
}

func appConfigRevisionTimestamp(seq uint64) string {
	if seq == 0 {
		seq = 1
	}
	return time.Date(2026, 6, 16, 0, 0, 0, 0, time.UTC).Add(time.Duration(seq-1) * time.Second).Format("2006-01-02T15:04:05.0000000Z")
}

func appConfigAcceptDatetime(req *http.Request) (time.Time, string, bool, *service.Response, error) {
	value := strings.TrimSpace(req.Header.Get("Accept-Datetime"))
	if value == "" {
		return time.Time{}, "", false, nil, nil
	}
	parsed, ok := parseAppConfigDatetime(value)
	if !ok {
		resp, err := appConfigProblemResponse(http.StatusBadRequest, "https://azconfig.io/errors/invalid-argument", "Invalid request header 'Accept-Datetime'.", "Accept-Datetime", "The Accept-Datetime header must be a valid HTTP-date or ISO-8601 timestamp.")
		return time.Time{}, "", false, resp, err
	}
	parsed = parsed.UTC()
	return parsed, parsed.Format(http.TimeFormat), true, nil, nil
}

func parseAppConfigDatetime(value string) (time.Time, bool) {
	for _, layout := range []string{http.TimeFormat, time.RFC1123, time.RFC1123Z, time.RFC3339Nano} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

func appConfigHTTPDate(value string) string {
	parsed, ok := parseAppConfigDatetime(value)
	if !ok {
		return value
	}
	return parsed.UTC().Format(http.TimeFormat)
}

func appConfigHistoricalKeyValues(revisions []keyValue, at time.Time) []keyValue {
	latest := make(map[string]keyValue)
	latestTimes := make(map[string]time.Time)
	for _, revision := range revisions {
		modified, ok := parseAppConfigDatetime(revision.LastModified)
		if !ok || modified.After(at) {
			continue
		}
		key := keyValueKey(revision.Key, revision.Label)
		if previous, exists := latestTimes[key]; exists && (modified.Before(previous) || modified.Equal(previous) && revision.ETag <= latest[key].ETag) {
			continue
		}
		latest[key] = revision
		latestTimes[key] = modified
	}
	values := make([]keyValue, 0, len(latest))
	for _, kv := range latest {
		values = append(values, kv)
	}
	return values
}

func appConfigOriginalLink(req *http.Request) string {
	query := req.URL.Query()
	for key := range query {
		if strings.EqualFold(key, "after") {
			delete(query, key)
		}
	}
	if encoded := query.Encode(); encoded != "" {
		return req.URL.EscapedPath() + "?" + encoded
	}
	return req.URL.EscapedPath()
}

func parseAppConfigItemRange(value string) (int, int, bool) {
	value = strings.TrimSpace(value)
	unit, span, ok := strings.Cut(value, "=")
	if !ok || strings.TrimSpace(strings.ToLower(unit)) != "items" {
		return 0, 0, false
	}
	startText, endText, ok := strings.Cut(strings.TrimSpace(span), "-")
	if !ok || strings.TrimSpace(startText) == "" || strings.TrimSpace(endText) == "" {
		return 0, 0, false
	}
	start, err := strconv.Atoi(strings.TrimSpace(startText))
	if err != nil || start < 0 {
		return 0, 0, false
	}
	end, err := strconv.Atoi(strings.TrimSpace(endText))
	if err != nil || end < start {
		return 0, 0, false
	}
	return start, end, true
}

func cloneKeyValue(kv keyValue) keyValue {
	kv.Tags = cloneAnyMap(kv.Tags)
	return kv
}

func cloneAnyMap(values map[string]any) map[string]any {
	if len(values) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func cloneStringMap(values map[string]string) map[string]string {
	if len(values) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(values))
	for key, value := range values {
		out[key] = value
	}
	return out
}

func (s *AppConfigurationService) preconditionFailedLocked(store string) (*service.Response, error) {
	return appConfigJSONResponse(http.StatusPreconditionFailed, map[string]any{
		"type":   "https://azconfig.io/errors/precondition-failed",
		"title":  "Precondition failed.",
		"status": http.StatusPreconditionFailed,
	}, problemJSONType, map[string]string{"Sync-Token": s.nextSyncTokenLocked(store)})
}

func (s *AppConfigurationService) nextSyncTokenLocked(store string) string {
	s.storeSyncSN[store]++
	sn := s.storeSyncSN[store]
	value := base64.StdEncoding.EncodeToString([]byte("0:" + strconv.FormatUint(sn, 10)))
	return "jtqGc1I4=" + value + ";sn=" + strconv.FormatUint(sn, 10)
}

func appConfigNotFoundResponse(key, syncToken string) (*service.Response, error) {
	return appConfigJSONResponse(http.StatusNotFound, map[string]any{
		"type":   "https://azconfig.io/errors/key-not-found",
		"title":  "The key does not exist.",
		"name":   key,
		"detail": "There is no value with the specified key and label.",
		"status": http.StatusNotFound,
	}, problemJSONType, map[string]string{"Sync-Token": syncToken})
}

func appConfigProblemResponse(status int, problemType, title, name, detail string) (*service.Response, error) {
	return appConfigProblemResponseWithHeaders(status, problemType, title, name, detail, nil)
}

func appConfigProblemResponseWithHeaders(status int, problemType, title, name, detail string, headers map[string]string) (*service.Response, error) {
	body := map[string]any{
		"type":   problemType,
		"title":  title,
		"name":   name,
		"detail": detail,
		"status": status,
	}
	return appConfigJSONResponse(status, body, problemJSONType, headers)
}

func appConfigJSONResponse(statusCode int, body any, contentType string, headers map[string]string) (*service.Response, error) {
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     statusCode,
		RawBody:        data,
		RawContentType: contentType,
		Headers:        headers,
	}, nil
}

func keyValueResponseBody(kv keyValue, selectFields []string) map[string]any {
	labelValue := any(nil)
	if kv.Label != "" {
		labelValue = kv.Label
	}
	body := map[string]any{
		"key":           kv.Key,
		"label":         labelValue,
		"value":         kv.Value,
		"content_type":  kv.ContentType,
		"tags":          kv.Tags,
		"etag":          kv.ETag,
		"last_modified": kv.LastModified,
		"locked":        kv.Locked,
	}
	if len(selectFields) == 0 {
		return body
	}
	projected := make(map[string]any, len(selectFields))
	for _, field := range selectFields {
		if value, ok := body[field]; ok {
			projected[field] = value
		}
	}
	return projected
}

func projectAppConfigFields(body map[string]any, selectFields []string) map[string]any {
	if len(selectFields) == 0 {
		return body
	}
	projected := make(map[string]any, len(selectFields))
	for _, field := range selectFields {
		if value, ok := body[field]; ok {
			projected[field] = value
		}
	}
	return projected
}

func dataPlaneStoreAndPath(req *http.Request) (string, []string, bool) {
	host := normalizedHost(req)
	if strings.HasSuffix(host, ".azconfig.io") {
		store := strings.TrimSuffix(host, ".azconfig.io")
		if store != "" {
			return store, splitPath(req.URL.EscapedPath()), true
		}
	}

	parts := splitPath(req.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-appconfig") {
		store := strings.TrimSuffix(parts[0], "-appconfig")
		if store == "" {
			store = parts[0]
		}
		return store, parts[1:], true
	}
	return "", nil, false
}

func storeID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.AppConfiguration/configurationStores/" + name
}

func storeKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func keyValueKey(key, label string) string {
	return key + "\x00" + label
}

func appConfigListETag(values []keyValue) string {
	var builder strings.Builder
	builder.WriteString(strconv.Itoa(len(values)))
	for _, kv := range values {
		builder.WriteString("\x00")
		builder.WriteString(kv.Key)
		builder.WriteString("\x00")
		builder.WriteString(kv.Label)
		builder.WriteString("\x00")
		builder.WriteString(kv.ETag)
	}
	return base64.RawURLEncoding.EncodeToString([]byte(builder.String()))
}

func normalizeLabel(label string) string {
	if label == "" || label == "\x00" {
		return ""
	}
	return label
}

func preconditionMatches(header, existingETag string, exists bool) bool {
	if header == "" {
		return true
	}
	clientETag := stripQuotes(header)
	if clientETag == "*" {
		return exists
	}
	return exists && clientETag == existingETag
}

func preconditionNoneMatchFails(header, existingETag string, exists bool) bool {
	if header == "" {
		return false
	}
	clientETag := stripQuotes(header)
	if clientETag == "*" {
		return exists
	}
	return exists && clientETag == existingETag
}

func stripQuotes(etag string) string {
	etag = strings.TrimSpace(etag)
	if len(etag) >= 2 && etag[0] == '"' && etag[len(etag)-1] == '"' {
		return etag[1 : len(etag)-1]
	}
	return etag
}

func quoteETag(etag string) string {
	return `"` + etag + `"`
}

func parseSelect(query url.Values) []string {
	raw := query.Get("$select")
	if raw == "" {
		raw = query.Get("$Select")
	}
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	fields := make([]string, 0, len(parts))
	for _, part := range parts {
		if field := strings.TrimSpace(part); field != "" {
			fields = append(fields, field)
		}
	}
	return fields
}

func matchesWildcard(value, filter string) bool {
	filters := appConfigNameFilterValues([]string{filter})
	if len(filters) == 0 {
		return true
	}
	return matchesAppConfigNameFilters(value, filters)
}

func matchesLabel(label string, filters []string) bool {
	for _, filter := range filters {
		options := appConfigNameFilterValues([]string{filter})
		if len(options) == 0 && filter == "" {
			return true
		}
		for _, option := range options {
			if option == "" || option == "*" {
				return true
			}
			if option == "\x00" {
				if label == "" {
					return true
				}
				continue
			}
			if matchesAppConfigNameFilter(label, option) {
				return true
			}
		}
	}
	return false
}

func matchesRevisionWildcard(value, filter string) bool {
	filters := snapshotListFilterValues([]string{filter})
	if len(filters) == 0 {
		return true
	}
	return matchesRevisionNameFilters(value, filters)
}

func matchesRevisionLabel(label string, filters []string) bool {
	for _, filter := range filters {
		options := snapshotListFilterValues([]string{filter})
		if len(options) == 0 && filter == "" {
			return true
		}
		if matchesRevisionNameFilters(label, options) {
			return true
		}
	}
	return false
}

func matchesRevisionNameFilters(value string, filters []string) bool {
	for _, filter := range filters {
		if matchesRevisionNameFilter(value, filter) {
			return true
		}
	}
	return false
}

func matchesRevisionNameFilter(value, filter string) bool {
	if filter == "" || filter == "*" {
		return true
	}
	if filter == "\x00" {
		return value == ""
	}

	leadingWildcard := strings.HasPrefix(filter, "*")
	trailingWildcard := strings.HasSuffix(filter, "*")
	switch {
	case leadingWildcard && trailingWildcard && len(filter) > 1:
		needle := unescapeSnapshotListFilterValue(strings.TrimSuffix(strings.TrimPrefix(filter, "*"), "*"))
		return strings.Contains(value, needle)
	case leadingWildcard:
		needle := unescapeSnapshotListFilterValue(strings.TrimPrefix(filter, "*"))
		return strings.HasSuffix(value, needle)
	case trailingWildcard:
		needle := unescapeSnapshotListFilterValue(strings.TrimSuffix(filter, "*"))
		return strings.HasPrefix(value, needle)
	default:
		return value == unescapeSnapshotListFilterValue(filter)
	}
}

func tagsMatch(tags map[string]any, filters []string) bool {
	if len(filters) == 0 {
		return true
	}
	for _, filter := range filters {
		if filter == "" {
			continue
		}
		key, value, ok := strings.Cut(filter, "=")
		if !ok {
			key = filter
			value = ""
		}
		value = unescapeAppConfigFilterLiteral(value)
		tagValue, exists := tags[key]
		if !exists {
			return false
		}
		if value == "\x00" {
			if tagValue != nil {
				return false
			}
			continue
		}
		if fmt.Sprint(tagValue) != value {
			return false
		}
	}
	return true
}

func unescapeAppConfigFilterLiteral(value string) string {
	var builder strings.Builder
	escaped := false
	for _, r := range value {
		if escaped {
			builder.WriteRune(r)
			escaped = false
			continue
		}
		if r == '\\' {
			escaped = true
			continue
		}
		builder.WriteRune(r)
	}
	if escaped {
		builder.WriteRune('\\')
	}
	return builder.String()
}

func segmentIndex(parts []string, segment string) int {
	for i, part := range parts {
		if strings.EqualFold(part, segment) {
			return i
		}
	}
	return -1
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

func setDefault(values map[string]any, key string, value any) {
	if _, ok := values[key]; !ok {
		values[key] = value
	}
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
