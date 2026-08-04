package storage

import (
	"encoding/base64"
	"encoding/xml"
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

// StorageService implements the Azure Storage control plane plus first-slice
// Blob and Queue data-plane APIs.
type StorageService struct {
	mu         sync.RWMutex
	accounts   map[string]StorageAccount
	keys       map[string][]StorageAccountKey
	blobProps  map[string][]byte
	containers map[string]map[string]blobContainer
	queues     map[string]map[string]queue
	queueProps map[string][]byte
	tables     map[string]map[string]table
	fileShares map[string]map[string]fileShare
	nextID     uint64
}

func New() *StorageService {
	return &StorageService{
		accounts:   make(map[string]StorageAccount),
		keys:       make(map[string][]StorageAccountKey),
		blobProps:  make(map[string][]byte),
		containers: make(map[string]map[string]blobContainer),
		queues:     make(map[string]map[string]queue),
		queueProps: make(map[string][]byte),
		tables:     make(map[string]map[string]table),
		fileShares: make(map[string]map[string]fileShare),
	}
}

func (s *StorageService) Name() string { return "Microsoft.Storage" }

func (s *StorageService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdate", Method: http.MethodPut, IAMAction: "azure:Microsoft.Storage/storageAccounts/write"},
		{Name: "Get", Method: http.MethodGet, IAMAction: "azure:Microsoft.Storage/storageAccounts/read"},
		{Name: "List", Method: http.MethodGet, IAMAction: "azure:Microsoft.Storage/storageAccounts/read"},
		{Name: "Delete", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Storage/storageAccounts/delete"},
		{Name: "listKeys", Method: http.MethodPost, IAMAction: "azure:Microsoft.Storage/storageAccounts/listKeys/action"},
		{Name: "regenerateKey", Method: http.MethodPost, IAMAction: "azure:Microsoft.Storage/storageAccounts/regenerateKey/action"},
	}
}

func (s *StorageService) HealthCheck() error { return nil }

func (s *StorageService) ServiceKeys() []routing.ServiceKey {
	return []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.Storage/storageAccounts", APIVersion: controlPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Storage/blobServices", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Storage/queueServices", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Storage/tableServices", APIVersion: dataPlaneAPIVersion},
		{Provider: routing.ProviderAzure, Service: "Microsoft.Storage/fileServices", APIVersion: dataPlaneAPIVersion},
	}
}

func (s *StorageService) SupportsTemplateResource(resource map[string]any) bool {
	return strings.EqualFold(stringValue(resource["type"]), "Microsoft.Storage/storageAccounts")
}

func (s *StorageService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("storage account template resource is missing name")
	}

	body := map[string]any{
		"location": resource["location"],
		"kind":     resource["kind"],
		"sku":      resource["sku"],
		"tags":     resource["tags"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}
	resp, err := s.createOrUpdateStorageAccount(subscriptionID, resourceGroup, name, data)
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *StorageService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	account, endpoint := dataPlaneAccount(ctx.RawRequest)
	switch endpoint {
	case "blob":
		return s.handleBlob(ctx, account)
	case "queue":
		return s.handleQueue(ctx, account)
	case "table":
		return s.handleTable(ctx, account)
	case "file":
		return s.handleFile(ctx, account)
	}
	return s.handleControlPlane(ctx)
}

func (s *StorageService) handleControlPlane(ctx *service.RequestContext) (*service.Response, error) {
	parts := splitPath(ctx.RawRequest.URL.EscapedPath())
	providerIndex := -1
	for i, part := range parts {
		if strings.EqualFold(part, "providers") {
			providerIndex = i
			break
		}
	}
	if providerIndex < 4 || providerIndex+2 >= len(parts) ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[providerIndex+1], "Microsoft.Storage") ||
		!strings.EqualFold(parts[providerIndex+2], "storageAccounts") {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The storage account route is not implemented.")
	}

	subscriptionID := parts[1]
	resourceGroup := parts[3]
	if len(parts) == providerIndex+3 && ctx.RawRequest.Method == http.MethodGet {
		return s.listStorageAccounts(subscriptionID, resourceGroup)
	}
	if len(parts) == providerIndex+5 && ctx.RawRequest.Method == http.MethodPost {
		name := parts[providerIndex+3]
		switch {
		case strings.EqualFold(parts[providerIndex+4], "listKeys"):
			return s.listStorageAccountKeys(subscriptionID, resourceGroup, name)
		case strings.EqualFold(parts[providerIndex+4], "regenerateKey"):
			return s.regenerateStorageAccountKey(subscriptionID, resourceGroup, name, ctx.Body)
		}
	}
	if len(parts) != providerIndex+4 {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The storage account route is not implemented.")
	}

	name := parts[providerIndex+3]
	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateStorageAccount(subscriptionID, resourceGroup, name, ctx.Body)
	case http.MethodGet:
		return s.getStorageAccount(subscriptionID, resourceGroup, name)
	case http.MethodDelete:
		return s.deleteStorageAccount(subscriptionID, resourceGroup, name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *StorageService) createOrUpdateStorageAccount(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location string         `json:"location"`
		Kind     string         `json:"kind"`
		SKU      StorageSKU     `json:"sku"`
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
	if input.Kind == "" {
		input.Kind = "StorageV2"
	}
	if input.SKU.Name == "" {
		input.SKU.Name = "Standard_LRS"
	}

	account := StorageAccount{
		ID:       storageAccountID(subscriptionID, resourceGroup, name),
		Name:     name,
		Type:     "Microsoft.Storage/storageAccounts",
		Location: input.Location,
		Tags:     stringifyTags(input.Tags),
		Kind:     input.Kind,
		SKU:      input.SKU,
		Properties: StorageAccountProperties{
			ProvisioningState: "Succeeded",
			PrimaryEndpoints: map[string]string{
				"blob":  "https://" + name + ".blob.core.windows.net/",
				"queue": "https://" + name + ".queue.core.windows.net/",
				"table": "https://" + name + ".table.core.windows.net/",
				"file":  "https://" + name + ".file.core.windows.net/",
			},
		},
	}

	s.mu.Lock()
	key := accountKey(subscriptionID, resourceGroup, name)
	s.accounts[key] = account
	if s.keys[key] == nil {
		s.keys[key] = initialStorageAccountKeys(key)
	}
	s.mu.Unlock()

	return azurearm.JSONResponse(http.StatusAccepted, account)
}

func (s *StorageService) getStorageAccount(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	account, ok := s.accounts[accountKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Storage account %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, account)
}

func (s *StorageService) listStorageAccounts(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := accountKeyPrefix(subscriptionID, resourceGroup)

	s.mu.RLock()
	values := make([]StorageAccount, 0)
	for key, account := range s.accounts {
		if strings.HasPrefix(key, prefix) {
			values = append(values, account)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *StorageService) deleteStorageAccount(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	key := accountKey(subscriptionID, resourceGroup, name)
	delete(s.accounts, key)
	delete(s.keys, key)
	delete(s.containers, strings.ToLower(name))
	delete(s.queues, strings.ToLower(name))
	s.mu.Unlock()
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *StorageService) listStorageAccountKeys(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	key := accountKey(subscriptionID, resourceGroup, name)

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.accounts[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Storage account %q could not be found.", name))
	}
	if s.keys[key] == nil {
		s.keys[key] = initialStorageAccountKeys(key)
	}
	return azurearm.JSONResponse(http.StatusOK, StorageAccountListKeysResult{Keys: cloneStorageAccountKeys(s.keys[key])})
}

func (s *StorageService) regenerateStorageAccountKey(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		KeyName string `json:"keyName"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
		}
	}
	if input.KeyName == "" {
		return azurearm.ErrorResponse(http.StatusBadRequest, "KeyNameRequired", "The storage account keyName is required.")
	}

	key := accountKey(subscriptionID, resourceGroup, name)
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.accounts[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Storage account %q could not be found.", name))
	}
	if s.keys[key] == nil {
		s.keys[key] = initialStorageAccountKeys(key)
	}
	for i := range s.keys[key] {
		if strings.EqualFold(s.keys[key][i].KeyName, input.KeyName) {
			s.nextID++
			s.keys[key][i].Value = storageAccountKeyValue(key, s.keys[key][i].KeyName, strconv.FormatUint(s.nextID, 10))
			return azurearm.JSONResponse(http.StatusOK, StorageAccountListKeysResult{Keys: cloneStorageAccountKeys(s.keys[key])})
		}
	}
	return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidKeyName", "The storage account keyName must be key1 or key2.")
}

func (s *StorageService) nextToken(prefix string) string {
	s.nextID++
	return prefix + "-" + strconv.FormatUint(s.nextID, 10)
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

func accountKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func accountKeyPrefix(subscriptionID, resourceGroup string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"
}

func storageAccountID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Storage/storageAccounts/" + name
}

func initialStorageAccountKeys(accountKey string) []StorageAccountKey {
	return []StorageAccountKey{
		{KeyName: "key1", Permissions: "Full", Value: storageAccountKeyValue(accountKey, "key1", "0")},
		{KeyName: "key2", Permissions: "Full", Value: storageAccountKeyValue(accountKey, "key2", "0")},
	}
}

func storageAccountKeyValue(accountKey, keyName, generation string) string {
	return base64.StdEncoding.EncodeToString([]byte("cloudmock:" + accountKey + ":" + keyName + ":" + generation))
}

func cloneStorageAccountKeys(keys []StorageAccountKey) []StorageAccountKey {
	out := make([]StorageAccountKey, len(keys))
	copy(out, keys)
	return out
}

func dataPlaneAccount(r *http.Request) (string, string) {
	host := strings.ToLower(r.Host)
	if host == "" {
		host = strings.ToLower(r.Header.Get("Host"))
	}
	if colon := strings.IndexByte(host, ':'); colon >= 0 {
		host = host[:colon]
	}
	for _, endpoint := range []string{"blob", "queue", "table", "file"} {
		for _, suffix := range []string{
			"." + endpoint + ".core.windows.net",
			"." + endpoint + ".core.usgovcloudapi.net",
			"." + endpoint + ".core.chinacloudapi.cn",
			"." + endpoint + ".core.cloudapi.de",
		} {
			if strings.HasSuffix(host, suffix) {
				return strings.TrimSuffix(host, suffix), endpoint
			}
		}
	}
	parts := splitPath(r.URL.EscapedPath())
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-queue") {
		account := strings.TrimSuffix(parts[0], "-queue")
		if account == "" {
			account = parts[0]
		}
		return account, "queue"
	}
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-table") {
		account := strings.TrimSuffix(parts[0], "-table")
		if account == "" {
			account = parts[0]
		}
		return account, "table"
	}
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-blob") {
		account := strings.TrimSuffix(parts[0], "-blob")
		if account == "" {
			account = parts[0]
		}
		return account, "blob"
	}
	if len(parts) > 0 && strings.HasSuffix(strings.ToLower(parts[0]), "-file") {
		account := strings.TrimSuffix(parts[0], "-file")
		if account == "" {
			account = parts[0]
		}
		return account, "file"
	}
	if len(parts) > 0 && isLocalStorageHost(host) && strings.EqualFold(parts[0], "devstoreaccount1") &&
		(len(parts) > 1 || isBlobAccountRootComp(r.URL.Query().Get("comp"))) {
		return parts[0], "blob"
	}
	return "", ""
}

func isBlobAccountRootComp(comp string) bool {
	return strings.EqualFold(comp, "list") ||
		strings.EqualFold(comp, "blobs") ||
		strings.EqualFold(comp, "properties") ||
		strings.EqualFold(comp, "userdelegationkey")
}

func isLocalStorageHost(host string) bool {
	switch host {
	case "localhost", "127.0.0.1":
		return true
	default:
		return false
	}
}

func metadataFromHeaders(header http.Header) map[string]string {
	metadata := make(map[string]string)
	for key, values := range header {
		if !strings.HasPrefix(strings.ToLower(key), "x-ms-meta-") || len(values) == 0 {
			continue
		}
		metadata[strings.TrimPrefix(strings.ToLower(key), "x-ms-meta-")] = values[0]
	}
	if len(metadata) == 0 {
		return nil
	}
	return metadata
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

func storageHeaders(etag string, lastModified time.Time) map[string]string {
	return map[string]string{
		"ETag":          etag,
		"Last-Modified": lastModified.UTC().Format(http.TimeFormat),
		"x-ms-version":  dataPlaneAPIVersion,
	}
}

func xmlResponse(statusCode int, body any) (*service.Response, error) {
	data, err := xml.Marshal(body)
	if err != nil {
		return nil, err
	}
	return &service.Response{
		StatusCode:     statusCode,
		RawBody:        data,
		RawContentType: "application/xml",
	}, nil
}

func emptyResponse(statusCode int, headers map[string]string) (*service.Response, error) {
	return &service.Response{StatusCode: statusCode, Headers: headers}, nil
}
