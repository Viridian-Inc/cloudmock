package storage_test

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"hash/crc64"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/azure/storage"
)

func storageCtx(t *testing.T, method, targetURL string, body []byte, headers map[string]string) *service.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, targetURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	return &service.RequestContext{
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       body,
	}
}

func storageCtxWithHeaders(t *testing.T, method, targetURL string, body []byte, headers http.Header) *service.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, targetURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	for k, values := range headers {
		for _, v := range values {
			req.Header.Add(k, v)
		}
	}
	return &service.RequestContext{
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       body,
	}
}

func decodeJSONBody(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	if resp.RawContentType != "application/json" {
		t.Fatalf("expected application/json, got %q", resp.RawContentType)
	}
	var out map[string]any
	if err := json.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("failed to decode JSON response: %v", err)
	}
	return out
}

type queueMessageForTest struct {
	MessageID    string `xml:"MessageId"`
	PopReceipt   string `xml:"PopReceipt"`
	Text         string `xml:"MessageText"`
	DequeueCount int    `xml:"DequeueCount"`
}

func decodeQueueMessages(t *testing.T, body []byte) []queueMessageForTest {
	t.Helper()
	var messages struct {
		Messages []queueMessageForTest `xml:"QueueMessage"`
	}
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&messages); err != nil && err != io.EOF {
		t.Fatalf("failed to decode queue messages: %v", err)
	}
	return messages.Messages
}

func looksLikeCanonicalUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i, c := range strings.ToLower(value) {
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			if !(c >= '0' && c <= '9') && !(c >= 'a' && c <= 'f') {
				return false
			}
		}
	}
	return true
}

func TestStorageAccountLifecycle(t *testing.T) {
	svc := storage.New()
	payload := []byte(`{"location":"westus2","kind":"StorageV2","sku":{"name":"Standard_LRS"},"tags":{"env":"test"}}`)

	createResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest?api-version=2024-01-01", payload, nil))
	if err != nil {
		t.Fatalf("create storage account returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected create status 202, got %d", createResp.StatusCode)
	}
	created := decodeJSONBody(t, createResp)
	if created["name"] != "acctest" {
		t.Errorf("unexpected account name: %v", created["name"])
	}
	props := created["properties"].(map[string]any)
	endpoints := props["primaryEndpoints"].(map[string]any)
	if endpoints["blob"] != "https://acctest.blob.core.windows.net/" {
		t.Errorf("unexpected blob endpoint: %v", endpoints["blob"])
	}
	if endpoints["queue"] != "https://acctest.queue.core.windows.net/" {
		t.Errorf("unexpected queue endpoint: %v", endpoints["queue"])
	}
	if endpoints["file"] != "https://acctest.file.core.windows.net/" {
		t.Errorf("unexpected file endpoint: %v", endpoints["file"])
	}

	getResp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest?api-version=2024-01-01", nil, nil))
	if err != nil {
		t.Fatalf("get storage account returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d", getResp.StatusCode)
	}

	listResp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts?api-version=2024-01-01", nil, nil))
	if err != nil {
		t.Fatalf("list storage accounts returned error: %v", err)
	}
	listed := decodeJSONBody(t, listResp)
	if got := len(listed["value"].([]any)); got != 1 {
		t.Fatalf("expected one storage account, got %d", got)
	}

	deleteResp, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest?api-version=2024-01-01", nil, nil))
	if err != nil {
		t.Fatalf("delete storage account returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete status 202, got %d", deleteResp.StatusCode)
	}
}

func TestStorageAccountListKeysAndRegenerateKey(t *testing.T) {
	svc := storage.New()
	createPayload := []byte(`{"location":"westus2","kind":"StorageV2","sku":{"name":"Standard_LRS"}}`)
	accountURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest?api-version=2024-01-01"
	_, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL, createPayload, nil))
	if err != nil {
		t.Fatalf("create storage account returned error: %v", err)
	}

	listKeysURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest/listKeys?api-version=2024-01-01"
	listResp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, listKeysURL, nil, nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list keys status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeJSONBody(t, listResp)
	keys := listed["keys"].([]any)
	if len(keys) != 2 {
		t.Fatalf("expected two storage keys, got %d", len(keys))
	}
	key1 := keys[0].(map[string]any)
	key2 := keys[1].(map[string]any)
	if key1["keyName"] != "key1" || key1["permissions"] != "Full" || key1["value"] == "" {
		t.Fatalf("unexpected key1 shape: %v", key1)
	}
	if key2["keyName"] != "key2" || key2["permissions"] != "Full" || key2["value"] == "" {
		t.Fatalf("unexpected key2 shape: %v", key2)
	}

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest/regenerateKey?api-version=2024-01-01"
	regenResp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, regenerateURL, []byte(`{"keyName":"key2"}`), nil))
	if err != nil {
		t.Fatalf("regenerate key returned error: %v", err)
	}
	if regenResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate key status 200, got %d; body=%s", regenResp.StatusCode, string(regenResp.RawBody))
	}
	regenerated := decodeJSONBody(t, regenResp)
	rotatedKeys := regenerated["keys"].([]any)
	rotatedKey1 := rotatedKeys[0].(map[string]any)
	rotatedKey2 := rotatedKeys[1].(map[string]any)
	if rotatedKey1["value"] != key1["value"] {
		t.Fatalf("expected key1 to remain stable, before=%v after=%v", key1["value"], rotatedKey1["value"])
	}
	if rotatedKey2["value"] == key2["value"] {
		t.Fatalf("expected key2 to rotate, still got %v", rotatedKey2["value"])
	}
}

func TestStorageServiceKeysIncludeTableDataPlane(t *testing.T) {
	svc := storage.New()

	seen := make(map[string]bool)
	for _, key := range svc.ServiceKeys() {
		seen[string(key.Provider)+"|"+key.Service+"|"+key.APIVersion] = true
	}

	if !seen["azure|Microsoft.Storage/tableServices|2023-11-03"] {
		t.Fatalf("expected table data-plane service key")
	}
}

func TestBlobServicePropertiesGetSetRoundTrip(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"

	defaultProperties, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "blob-props-default",
	}))
	if err != nil {
		t.Fatalf("get default blob service properties returned error: %v", err)
	}
	if defaultProperties.StatusCode != http.StatusOK || defaultProperties.RawContentType != "application/xml" {
		t.Fatalf("expected default blob service properties status 200 XML, got status=%d contentType=%q body=%s", defaultProperties.StatusCode, defaultProperties.RawContentType, string(defaultProperties.RawBody))
	}
	if defaultProperties.Headers["x-ms-version"] != "2023-11-03" || defaultProperties.Headers["x-ms-client-request-id"] != "blob-props-default" || defaultProperties.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected version, client request ID, and request ID headers, got %v", defaultProperties.Headers)
	}
	defaultBody := string(defaultProperties.RawBody)
	for _, fragment := range []string{"<StorageServiceProperties>", "<Logging>", "<HourMetrics>", "<MinuteMetrics>", "<Cors>", "<DefaultServiceVersion>", "<DeleteRetentionPolicy>", "<StaticWebsite>"} {
		if !strings.Contains(defaultBody, fragment) {
			t.Fatalf("expected default blob service properties to include %q, got %s", fragment, defaultBody)
		}
	}

	customProperties := []byte(`<StorageServiceProperties>
  <DefaultServiceVersion>2023-11-03</DefaultServiceVersion>
  <StaticWebsite>
    <Enabled>true</Enabled>
    <DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath>
    <ErrorDocument404Path>error/404.html</ErrorDocument404Path>
  </StaticWebsite>
</StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, customProperties, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "blob-props-set",
	}))
	if err != nil {
		t.Fatalf("set blob service properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted || len(setProperties.RawBody) != 0 {
		t.Fatalf("expected set blob service properties status 202 without body, got status=%d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}
	if setProperties.Headers["x-ms-version"] != "2023-11-03" || setProperties.Headers["x-ms-client-request-id"] != "blob-props-set" || setProperties.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected set response to echo version and client request ID, got %v", setProperties.Headers)
	}

	updatedProperties, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get updated blob service properties returned error: %v", err)
	}
	updatedBody := string(updatedProperties.RawBody)
	for _, fragment := range []string{
		"<Logging>",
		"<HourMetrics>",
		"<MinuteMetrics>",
		"<Cors>",
		"<DefaultServiceVersion>2023-11-03</DefaultServiceVersion>",
		"<Enabled>true</Enabled>",
		"<DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath>",
		"<ErrorDocument404Path>error/404.html</ErrorDocument404Path>",
	} {
		if !strings.Contains(updatedBody, fragment) {
			t.Fatalf("expected updated blob service properties to preserve/include %q, got %s", fragment, updatedBody)
		}
	}
}

func TestBlobCORSPreflightUsesStoredServiceProperties(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>GET,PUT</AllowedMethods><MaxAgeInSeconds>300</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*,x-ms-request-id</ExposedHeaders><AllowedHeaders>x-ms-client-request-id,x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set blob service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set blob service CORS properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	preflight, err := svc.HandleRequest(storageCtx(t, http.MethodOptions, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version":                   "2023-11-03",
		"Origin":                         "https://app.example",
		"Access-Control-Request-Method":  "GET",
		"Access-Control-Request-Headers": "x-ms-client-request-id,x-ms-meta-target-id",
		"x-ms-client-request-id":         "blob-cors-preflight",
	}))
	if err != nil {
		t.Fatalf("blob CORS preflight returned error: %v", err)
	}
	if preflight.StatusCode != http.StatusOK {
		t.Fatalf("expected matching blob CORS preflight status 200, got %d body=%s", preflight.StatusCode, string(preflight.RawBody))
	}
	if preflight.Headers["Access-Control-Allow-Origin"] != "https://app.example" {
		t.Fatalf("expected allow-origin header for matching origin, got %q", preflight.Headers["Access-Control-Allow-Origin"])
	}
	if preflight.Headers["Access-Control-Allow-Methods"] != "GET" {
		t.Fatalf("expected allow-methods header for requested method, got %q", preflight.Headers["Access-Control-Allow-Methods"])
	}
	if preflight.Headers["Access-Control-Allow-Headers"] != "x-ms-client-request-id,x-ms-meta-target-id" {
		t.Fatalf("expected allow-headers header for requested headers, got %q", preflight.Headers["Access-Control-Allow-Headers"])
	}
	if preflight.Headers["Access-Control-Max-Age"] != "300" {
		t.Fatalf("expected max-age from CORS rule, got %q", preflight.Headers["Access-Control-Max-Age"])
	}
	if preflight.Headers["x-ms-client-request-id"] != "blob-cors-preflight" || preflight.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected blob CORS preflight to echo request IDs, got %v", preflight.Headers)
	}
}

func TestBlobCORSActualGetAddsHeadersForMatchingStoredRule(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>300</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*,x-ms-request-id</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	_, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set blob service CORS properties returned error: %v", err)
	}
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("cors body"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"x-ms-meta-owner": "docs",
	}))

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"Origin":                 "https://app.example",
		"x-ms-client-request-id": "blob-cors-actual",
	}))
	if err != nil {
		t.Fatalf("blob CORS actual GET returned error: %v", err)
	}
	if getBlob.StatusCode != http.StatusOK || string(getBlob.RawBody) != "cors body" {
		t.Fatalf("expected normal blob GET response with body, got status=%d body=%s", getBlob.StatusCode, string(getBlob.RawBody))
	}
	if getBlob.Headers["Access-Control-Allow-Origin"] != "https://app.example" {
		t.Fatalf("expected allow-origin header for matching actual CORS request, got %q", getBlob.Headers["Access-Control-Allow-Origin"])
	}
	if getBlob.Headers["Access-Control-Expose-Headers"] != "x-ms-meta-data*,x-ms-request-id" {
		t.Fatalf("expected expose-headers from matching CORS rule, got %q", getBlob.Headers["Access-Control-Expose-Headers"])
	}
	if getBlob.Headers["Vary"] != "Origin" {
		t.Fatalf("expected Vary Origin for exact-origin GET CORS match, got %q", getBlob.Headers["Vary"])
	}
	if getBlob.Headers["x-ms-client-request-id"] != "blob-cors-actual" {
		t.Fatalf("expected blob GET to preserve client request ID, got %v", getBlob.Headers)
	}
}

func TestBlobServicePropertiesRejectsInvalidCorsRules(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET,PATCH</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid blob service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid blob service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	tooManyCors := []byte(`<StorageServiceProperties><Cors>
<CorsRule><AllowedOrigins>https://one.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://two.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://three.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://four.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://five.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://six.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
</Cors></StorageServiceProperties>`)
	setTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooManyCors, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too many blob service CORS rules returned error: %v", err)
	}
	if setTooMany.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many blob service CORS rules status 400, got %d body=%s", setTooMany.StatusCode, string(setTooMany.RawBody))
	}

	missingCorsField := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://missing.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setMissingField, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, missingCorsField, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set missing-field blob service CORS rule returned error: %v", err)
	}
	if setMissingField.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing CORS field status 400, got %d body=%s", setMissingField.StatusCode, string(setMissingField.RawBody))
	}

	unsupportedMethod := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://trace.example</AllowedOrigins><AllowedMethods>GET,TRACE</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setUnsupported, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, unsupportedMethod, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set unsupported-method blob service CORS rule returned error: %v", err)
	}
	if setUnsupported.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unsupported CORS method status 400, got %d body=%s", setUnsupported.StatusCode, string(setUnsupported.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob service properties after invalid CORS updates returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") ||
		strings.Contains(body, "https://six.example") ||
		strings.Contains(body, "https://missing.example") ||
		strings.Contains(body, "https://trace.example") {
		t.Fatalf("expected invalid CORS updates not to replace valid properties, got %s", body)
	}
}

func TestBlobServicePropertiesRejectsInvalidRootProperties(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties>
  <DefaultServiceVersion>2023-11-03</DefaultServiceVersion>
  <DeleteRetentionPolicy><Enabled>true</Enabled><Days>7</Days></DeleteRetentionPolicy>
  <StaticWebsite><Enabled>true</Enabled><DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath><ErrorDocument404Path>error/404.html</ErrorDocument404Path></StaticWebsite>
</StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid blob service root properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid blob service root properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	invalidUpdates := []struct {
		name string
		body []byte
	}{
		{
			name: "enabled hour metrics missing IncludeAPIs",
			body: []byte(`<StorageServiceProperties><HourMetrics><Version>1.0</Version><Enabled>true</Enabled><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></HourMetrics></StorageServiceProperties>`),
		},
		{
			name: "delete retention days too high",
			body: []byte(`<StorageServiceProperties><DeleteRetentionPolicy><Enabled>true</Enabled><Days>366</Days></DeleteRetentionPolicy></StorageServiceProperties>`),
		},
		{
			name: "static website has mutually exclusive index roots",
			body: []byte(`<StorageServiceProperties><StaticWebsite><Enabled>true</Enabled><IndexDocument>index.html</IndexDocument><DefaultIndexDocumentPath>default.html</DefaultIndexDocumentPath></StaticWebsite></StorageServiceProperties>`),
		},
	}
	for _, update := range invalidUpdates {
		setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, update.body, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("set invalid blob service properties %q returned error: %v", update.name, err)
		}
		if setInvalid.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid blob service properties %q status 400, got %d body=%s", update.name, setInvalid.StatusCode, string(setInvalid.RawBody))
		}
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob service properties after invalid root updates returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	for _, fragment := range []string{
		"<DefaultServiceVersion>2023-11-03</DefaultServiceVersion>",
		"<Days>7</Days>",
		"<DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath>",
		"<ErrorDocument404Path>error/404.html</ErrorDocument404Path>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected invalid root updates to preserve valid fragment %q, got %s", fragment, body)
		}
	}
	for _, forbidden := range []string{"<Days>366</Days>", "<IndexDocument>index.html</IndexDocument>", "<DefaultIndexDocumentPath>default.html</DefaultIndexDocumentPath>"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected invalid root updates not to store %q, got %s", forbidden, body)
		}
	}
}

func TestBlobServicePropertiesRejectsRootsUnsupportedByAPIVersion(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties>
  <DefaultServiceVersion>2023-11-03</DefaultServiceVersion>
  <DeleteRetentionPolicy><Enabled>true</Enabled><Days>7</Days></DeleteRetentionPolicy>
  <StaticWebsite><Enabled>true</Enabled><DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath></StaticWebsite>
</StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid modern blob service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid modern blob service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	unsupportedUpdates := []struct {
		name       string
		apiVersion string
		body       []byte
	}{
		{
			name:       "CORS before 2013-08-15",
			apiVersion: "2012-02-12",
			body:       []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://old.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-request-id</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`),
		},
		{
			name:       "DeleteRetentionPolicy before 2017-07-29",
			apiVersion: "2016-05-31",
			body:       []byte(`<StorageServiceProperties><DeleteRetentionPolicy><Enabled>true</Enabled><Days>3</Days></DeleteRetentionPolicy></StorageServiceProperties>`),
		},
		{
			name:       "StaticWebsite before 2018-03-28",
			apiVersion: "2017-07-29",
			body:       []byte(`<StorageServiceProperties><StaticWebsite><Enabled>true</Enabled><DefaultIndexDocumentPath>legacy.html</DefaultIndexDocumentPath></StaticWebsite></StorageServiceProperties>`),
		},
	}
	for _, update := range unsupportedUpdates {
		setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, update.body, map[string]string{
			"x-ms-version": update.apiVersion,
		}))
		if err != nil {
			t.Fatalf("set unsupported blob service properties %q returned error: %v", update.name, err)
		}
		if setInvalid.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected unsupported blob service properties %q status 400, got %d body=%s", update.name, setInvalid.StatusCode, string(setInvalid.RawBody))
		}
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob service properties after unsupported-version updates returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	for _, fragment := range []string{
		"<DefaultServiceVersion>2023-11-03</DefaultServiceVersion>",
		"<Days>7</Days>",
		"<DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected unsupported-version updates to preserve valid fragment %q, got %s", fragment, body)
		}
	}
	for _, forbidden := range []string{"https://old.example", "<Days>3</Days>", "<DefaultIndexDocumentPath>legacy.html</DefaultIndexDocumentPath>"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected unsupported-version updates not to store %q, got %s", forbidden, body)
		}
	}
}

func TestBlobServicePropertiesGetProjectsLegacyMetricsForOlderVersions(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	modernProperties := []byte(`<StorageServiceProperties>
  <HourMetrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>true</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>5</Days></RetentionPolicy></HourMetrics>
  <MinuteMetrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>6</Days></RetentionPolicy></MinuteMetrics>
  <Cors><CorsRule><AllowedOrigins>https://modern.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-request-id</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors>
  <DefaultServiceVersion>2023-11-03</DefaultServiceVersion>
  <DeleteRetentionPolicy><Enabled>true</Enabled><Days>7</Days></DeleteRetentionPolicy>
  <StaticWebsite><Enabled>true</Enabled><DefaultIndexDocumentPath>index.html</DefaultIndexDocumentPath></StaticWebsite>
</StorageServiceProperties>`)
	setModern, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, modernProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern blob service properties returned error: %v", err)
	}
	if setModern.StatusCode != http.StatusAccepted {
		t.Fatalf("expected modern blob service properties status 202, got %d body=%s", setModern.StatusCode, string(setModern.RawBody))
	}

	legacy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("get legacy blob service properties returned error: %v", err)
	}
	if legacy.StatusCode != http.StatusOK {
		t.Fatalf("expected legacy blob service properties status 200, got %d body=%s", legacy.StatusCode, string(legacy.RawBody))
	}
	if legacy.Headers["x-ms-version"] != "2012-02-12" {
		t.Fatalf("expected legacy response to echo x-ms-version 2012-02-12, got %q", legacy.Headers["x-ms-version"])
	}
	body := string(legacy.RawBody)
	for _, fragment := range []string{
		"<Logging>",
		"<Metrics>",
		"<Days>5</Days>",
		"<DefaultServiceVersion>2023-11-03</DefaultServiceVersion>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected legacy blob service properties to include %q, got %s", fragment, body)
		}
	}
	for _, forbidden := range []string{"<HourMetrics>", "<MinuteMetrics>", "<Cors>", "<DeleteRetentionPolicy>", "<StaticWebsite>", "https://modern.example", "<Days>6</Days>", "<Days>7</Days>"} {
		if strings.Contains(body, forbidden) {
			t.Fatalf("expected legacy blob service properties to omit %q, got %s", forbidden, body)
		}
	}
}

func TestBlobServicePropertiesGetProjectsModernRootsByAPIVersion(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	modernProperties := []byte(`<StorageServiceProperties>
  <HourMetrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>true</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>5</Days></RetentionPolicy></HourMetrics>
  <MinuteMetrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>6</Days></RetentionPolicy></MinuteMetrics>
  <Cors><CorsRule><AllowedOrigins>https://modern.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-request-id</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors>
  <DefaultServiceVersion>2023-11-03</DefaultServiceVersion>
  <DeleteRetentionPolicy><Enabled>true</Enabled><Days>7</Days></DeleteRetentionPolicy>
  <StaticWebsite><Enabled>true</Enabled><DefaultIndexDocumentPath>default.html</DefaultIndexDocumentPath><ErrorDocument404Path>error/404.html</ErrorDocument404Path></StaticWebsite>
</StorageServiceProperties>`)
	setModern, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, modernProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern blob service properties returned error: %v", err)
	}
	if setModern.StatusCode != http.StatusAccepted {
		t.Fatalf("expected modern blob service properties status 202, got %d body=%s", setModern.StatusCode, string(setModern.RawBody))
	}

	cases := []struct {
		apiVersion string
		include    []string
		omit       []string
	}{
		{
			apiVersion: "2013-08-15",
			include:    []string{"<HourMetrics>", "<MinuteMetrics>", "<Cors>", "<DefaultServiceVersion>2023-11-03</DefaultServiceVersion>", "https://modern.example"},
			omit:       []string{"<DeleteRetentionPolicy>", "<StaticWebsite>", "<Days>7</Days>", "default.html", "error/404.html"},
		},
		{
			apiVersion: "2017-07-29",
			include:    []string{"<HourMetrics>", "<MinuteMetrics>", "<Cors>", "<DeleteRetentionPolicy>", "<Days>7</Days>"},
			omit:       []string{"<StaticWebsite>", "default.html", "error/404.html"},
		},
		{
			apiVersion: "2018-03-28",
			include:    []string{"<DeleteRetentionPolicy>", "<StaticWebsite>", "<ErrorDocument404Path>error/404.html</ErrorDocument404Path>"},
			omit:       []string{"<DefaultIndexDocumentPath>default.html</DefaultIndexDocumentPath>"},
		},
		{
			apiVersion: "2019-12-12",
			include:    []string{"<StaticWebsite>", "<DefaultIndexDocumentPath>default.html</DefaultIndexDocumentPath>", "<ErrorDocument404Path>error/404.html</ErrorDocument404Path>"},
		},
	}
	for _, tc := range cases {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
			"x-ms-version": tc.apiVersion,
		}))
		if err != nil {
			t.Fatalf("get blob service properties for %s returned error: %v", tc.apiVersion, err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("expected blob service properties status 200 for %s, got %d body=%s", tc.apiVersion, resp.StatusCode, string(resp.RawBody))
		}
		if resp.Headers["x-ms-version"] != tc.apiVersion {
			t.Fatalf("expected blob service properties response to echo version %s, got %q", tc.apiVersion, resp.Headers["x-ms-version"])
		}
		body := string(resp.RawBody)
		for _, fragment := range tc.include {
			if !strings.Contains(body, fragment) {
				t.Fatalf("expected %s blob service properties to include %q, got %s", tc.apiVersion, fragment, body)
			}
		}
		for _, forbidden := range tc.omit {
			if strings.Contains(body, forbidden) {
				t.Fatalf("expected %s blob service properties to omit %q, got %s", tc.apiVersion, forbidden, body)
			}
		}
	}
}

func TestBlobServicePropertiesModernGetAfterLegacySetUsesModernMetricsShape(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.blob.core.windows.net/?restype=service&comp=properties"
	legacyProperties := []byte(`<StorageServiceProperties>
  <Logging><Version>1.0</Version><Delete>true</Delete><Read>false</Read><Write>true</Write><RetentionPolicy><Enabled>true</Enabled><Days>3</Days></RetentionPolicy></Logging>
  <Metrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>true</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>11</Days></RetentionPolicy></Metrics>
  <DefaultServiceVersion>2012-02-12</DefaultServiceVersion>
</StorageServiceProperties>`)
	setLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, legacyProperties, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("set legacy blob service properties returned error: %v", err)
	}
	if setLegacy.StatusCode != http.StatusAccepted {
		t.Fatalf("expected legacy blob service properties status 202, got %d body=%s", setLegacy.StatusCode, string(setLegacy.RawBody))
	}

	modern, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get modern blob service properties after legacy set returned error: %v", err)
	}
	if modern.StatusCode != http.StatusOK {
		t.Fatalf("expected modern blob service properties status 200, got %d body=%s", modern.StatusCode, string(modern.RawBody))
	}
	body := string(modern.RawBody)
	for _, fragment := range []string{
		"<Logging>",
		"<Delete>true</Delete>",
		"<Write>true</Write>",
		"<HourMetrics>",
		"<Days>11</Days>",
		"<MinuteMetrics>",
		"<Cors>",
		"<DefaultServiceVersion>2012-02-12</DefaultServiceVersion>",
		"<DeleteRetentionPolicy>",
		"<StaticWebsite>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected modern blob service properties after legacy set to include %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "<Metrics>") {
		t.Fatalf("expected modern blob service properties after legacy set not to include legacy Metrics root, got %s", body)
	}
}

func TestBlobServiceStatsRequiresSecondaryEndpoint(t *testing.T) {
	svc := storage.New()

	primary, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/?restype=service&comp=stats", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get primary blob service stats returned error: %v", err)
	}
	if primary.StatusCode != http.StatusForbidden {
		t.Fatalf("expected primary blob service stats status 403, got %d body=%s", primary.StatusCode, string(primary.RawBody))
	}
	if !strings.Contains(string(primary.RawBody), "InsufficientAccountPermissions") {
		t.Fatalf("expected primary blob service stats to return InsufficientAccountPermissions, got %s", string(primary.RawBody))
	}

	secondary, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest-secondary.blob.core.windows.net/?restype=service&comp=stats", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "blob-stats-secondary",
	}))
	if err != nil {
		t.Fatalf("get secondary blob service stats returned error: %v", err)
	}
	if secondary.StatusCode != http.StatusOK || secondary.RawContentType != "application/xml" {
		t.Fatalf("expected secondary blob service stats status 200 XML, got status=%d contentType=%q body=%s", secondary.StatusCode, secondary.RawContentType, string(secondary.RawBody))
	}
	if secondary.Headers["x-ms-version"] != "2023-11-03" || secondary.Headers["x-ms-client-request-id"] != "blob-stats-secondary" || secondary.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected blob service stats response to echo request headers, got %v", secondary.Headers)
	}
	body := string(secondary.RawBody)
	for _, fragment := range []string{"<StorageServiceStats>", "<GeoReplication>", "<Status>live</Status>", "<LastSyncTime>"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected secondary blob service stats body to include %q, got %s", fragment, body)
		}
	}
}

func TestBlobUserDelegationKeyReturnsSignedKey(t *testing.T) {
	svc := storage.New()
	start := time.Now().UTC().Add(time.Hour).Truncate(time.Second).Format(time.RFC3339)
	expiry := time.Now().UTC().Add(2 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	body := []byte(`<KeyInfo><Start>` + start + `</Start><Expiry>` + expiry + `</Expiry><DelegatedUserTid>11111111-2222-3333-4444-555555555555</DelegatedUserTid></KeyInfo>`)

	resp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.blob.core.windows.net/?restype=service&comp=userdelegationkey", body, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "user-delegation-key",
	}))
	if err != nil {
		t.Fatalf("get user delegation key returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK || resp.RawContentType != "application/xml" {
		t.Fatalf("expected user delegation key status 200 XML, got status=%d contentType=%q body=%s", resp.StatusCode, resp.RawContentType, string(resp.RawBody))
	}
	if resp.Headers["x-ms-version"] != "2023-11-03" || resp.Headers["x-ms-client-request-id"] != "user-delegation-key" || resp.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected user delegation key response to echo request headers, got %v", resp.Headers)
	}
	responseBody := string(resp.RawBody)
	for _, fragment := range []string{
		"<UserDelegationKey>",
		"<SignedOid>00000000-0000-0000-0000-000000000001</SignedOid>",
		"<SignedTid>00000000-0000-0000-0000-000000000001</SignedTid>",
		"<SignedStart>" + start + "</SignedStart>",
		"<SignedExpiry>" + expiry + "</SignedExpiry>",
		"<SignedService>b</SignedService>",
		"<SignedVersion>2023-11-03</SignedVersion>",
		"<SignedDelegatedUserTid>11111111-2222-3333-4444-555555555555</SignedDelegatedUserTid>",
		"<Value>",
	} {
		if !strings.Contains(responseBody, fragment) {
			t.Fatalf("expected user delegation key response to include %q, got %s", fragment, responseBody)
		}
	}
}

func TestBlobDataPlaneLifecycle(t *testing.T) {
	svc := storage.New()

	createContainer, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-meta-team": "platform",
	}))
	if err != nil {
		t.Fatalf("create container returned error: %v", err)
	}
	if createContainer.StatusCode != http.StatusCreated {
		t.Fatalf("expected create container status 201, got %d", createContainer.StatusCode)
	}
	if createContainer.Headers["ETag"] == "" {
		t.Fatal("expected container ETag header")
	}

	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("hello azure"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "docs",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob returned error: %v", err)
	}
	if string(getBlob.RawBody) != "hello azure" {
		t.Fatalf("unexpected blob body: %q", string(getBlob.RawBody))
	}
	if getBlob.RawContentType != "text/plain" {
		t.Fatalf("unexpected blob content type: %q", getBlob.RawContentType)
	}
	if getBlob.Headers["x-ms-meta-owner"] != "docs" {
		t.Fatalf("expected blob metadata header, got %q", getBlob.Headers["x-ms-meta-owner"])
	}

	listBlobs, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs?restype=container&comp=list", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list blobs returned error: %v", err)
	}
	if !strings.Contains(string(listBlobs.RawBody), "<Name>readme.txt</Name>") {
		t.Fatalf("list blobs response did not include blob name: %s", string(listBlobs.RawBody))
	}

	deleteBlob, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete blob returned error: %v", err)
	}
	if deleteBlob.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete blob status 202, got %d", deleteBlob.StatusCode)
	}
}

func TestBlobDataPlaneLocalRouteContainerConflictAndDelete(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	blobURL := "http://localhost:4577/devstoreaccount1/docs/readme.txt"

	createContainer, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, map[string]string{
		"x-ms-meta-team": "platform",
	}))
	if err != nil {
		t.Fatalf("create local container returned error: %v", err)
	}
	if createContainer.StatusCode != http.StatusCreated {
		t.Fatalf("expected local create container status 201, got %d; body=%s", createContainer.StatusCode, string(createContainer.RawBody))
	}

	recreateContainer, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("recreate local container returned error: %v", err)
	}
	if recreateContainer.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate local container create status 409, got %d; body=%s", recreateContainer.StatusCode, string(recreateContainer.RawBody))
	}

	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL, []byte("local blob"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "compat",
	}))
	if err != nil {
		t.Fatalf("put local blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected local put blob status 201, got %d; body=%s", putBlob.StatusCode, string(putBlob.RawBody))
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, blobURL, nil, nil))
	if err != nil {
		t.Fatalf("get local blob returned error: %v", err)
	}
	if getBlob.StatusCode != http.StatusOK || string(getBlob.RawBody) != "local blob" || getBlob.Headers["x-ms-meta-owner"] != "compat" {
		t.Fatalf("expected local blob body and metadata, status=%d headers=%v body=%q", getBlob.StatusCode, getBlob.Headers, string(getBlob.RawBody))
	}

	deleteContainer, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("delete local container returned error: %v", err)
	}
	if deleteContainer.StatusCode != http.StatusAccepted {
		t.Fatalf("expected local delete container status 202, got %d; body=%s", deleteContainer.StatusCode, string(deleteContainer.RawBody))
	}

	getDeletedBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, blobURL, nil, nil))
	if err != nil {
		t.Fatalf("get deleted local blob returned error: %v", err)
	}
	if getDeletedBlob.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted local blob status 404, got %d", getDeletedBlob.StatusCode)
	}
}

func TestBlobDataPlaneConditionalGetAndDeleteHonorETags(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("conditional data"), map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}
	etag := putBlob.Headers["ETag"]
	if etag == "" {
		t.Fatal("expected uploaded blob ETag")
	}

	matchedGet, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"If-Match":     etag,
	}))
	if err != nil {
		t.Fatalf("conditional get returned error: %v", err)
	}
	if matchedGet.StatusCode != http.StatusOK {
		t.Fatalf("expected matching If-Match get status 200, got %d; body=%s", matchedGet.StatusCode, string(matchedGet.RawBody))
	}

	staleGet, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"If-Match":     `"wrong-etag"`,
	}))
	if err != nil {
		t.Fatalf("stale conditional get returned error: %v", err)
	}
	if staleGet.StatusCode != http.StatusPreconditionFailed || staleGet.Headers["x-ms-error-code"] != "ConditionNotMet" {
		t.Fatalf("expected stale If-Match get 412 ConditionNotMet, got %d headers=%v body=%s", staleGet.StatusCode, staleGet.Headers, string(staleGet.RawBody))
	}

	blockedDelete, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"If-None-Match": etag,
	}))
	if err != nil {
		t.Fatalf("conditional delete returned error: %v", err)
	}
	if blockedDelete.StatusCode != http.StatusPreconditionFailed || blockedDelete.Headers["x-ms-error-code"] != "ConditionNotMet" {
		t.Fatalf("expected matching If-None-Match delete 412 ConditionNotMet, got %d headers=%v body=%s", blockedDelete.StatusCode, blockedDelete.Headers, string(blockedDelete.RawBody))
	}

	stillExists, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get after blocked delete returned error: %v", err)
	}
	if stillExists.StatusCode != http.StatusOK || string(stillExists.RawBody) != "conditional data" {
		t.Fatalf("expected blob to survive blocked conditional delete, got %d body=%s", stillExists.StatusCode, string(stillExists.RawBody))
	}
}

func TestBlobDataPlaneConditionalGetHonorsDateHeaders(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/dates.txt", []byte("dated data"), map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}
	etag := putBlob.Headers["ETag"]
	if etag == "" {
		t.Fatal("expected uploaded blob ETag")
	}
	lastModified, err := time.Parse(http.TimeFormat, putBlob.Headers["Last-Modified"])
	if err != nil {
		t.Fatalf("expected parseable Last-Modified header, got %q: %v", putBlob.Headers["Last-Modified"], err)
	}
	beforeLastModified := lastModified.Add(-time.Minute).UTC().Format(http.TimeFormat)
	atLastModified := lastModified.UTC().Format(http.TimeFormat)

	modified, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/dates.txt", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("modified-since get returned error: %v", err)
	}
	if modified.StatusCode != http.StatusOK || string(modified.RawBody) != "dated data" {
		t.Fatalf("expected older If-Modified-Since to allow read, got %d body=%s", modified.StatusCode, string(modified.RawBody))
	}

	notModified, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/dates.txt", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": atLastModified,
	}))
	if err != nil {
		t.Fatalf("not-modified get returned error: %v", err)
	}
	if notModified.StatusCode != http.StatusNotModified || len(notModified.RawBody) != 0 {
		t.Fatalf("expected If-Modified-Since at Last-Modified to return 304 without a body, got %d body=%s", notModified.StatusCode, string(notModified.RawBody))
	}

	noneMatch, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/dates.txt", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"If-None-Match": etag,
	}))
	if err != nil {
		t.Fatalf("none-match get returned error: %v", err)
	}
	if noneMatch.StatusCode != http.StatusNotModified || len(noneMatch.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match on read to return 304 without a body, got %d body=%s", noneMatch.StatusCode, string(noneMatch.RawBody))
	}

	unmodifiedFailed, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/dates.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"If-Unmodified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("unmodified-since get returned error: %v", err)
	}
	if unmodifiedFailed.StatusCode != http.StatusPreconditionFailed || unmodifiedFailed.Headers["x-ms-error-code"] != "ConditionNotMet" {
		t.Fatalf("expected older If-Unmodified-Since to fail 412 ConditionNotMet, got %d headers=%v body=%s", unmodifiedFailed.StatusCode, unmodifiedFailed.Headers, string(unmodifiedFailed.RawBody))
	}
}

func TestBlobPropertiesConditionalReadHonorsDateHeaders(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/properties.txt", []byte("properties data"), map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}
	etag := putBlob.Headers["ETag"]
	if etag == "" {
		t.Fatal("expected uploaded blob ETag")
	}
	lastModified, err := time.Parse(http.TimeFormat, putBlob.Headers["Last-Modified"])
	if err != nil {
		t.Fatalf("expected parseable Last-Modified header, got %q: %v", putBlob.Headers["Last-Modified"], err)
	}
	beforeLastModified := lastModified.Add(-time.Minute).UTC().Format(http.TimeFormat)
	atLastModified := lastModified.UTC().Format(http.TimeFormat)

	properties, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "https://acctest.blob.core.windows.net/docs/properties.txt", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("conditional blob properties returned error: %v", err)
	}
	if properties.StatusCode != http.StatusOK || len(properties.RawBody) != 0 || properties.Headers["Content-Length"] != "15" {
		t.Fatalf("expected older If-Modified-Since to return properties without a body, got status=%d headers=%v body=%s", properties.StatusCode, properties.Headers, string(properties.RawBody))
	}

	notModified, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "https://acctest.blob.core.windows.net/docs/properties.txt", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": atLastModified,
	}))
	if err != nil {
		t.Fatalf("not-modified blob properties returned error: %v", err)
	}
	if notModified.StatusCode != http.StatusNotModified || len(notModified.RawBody) != 0 {
		t.Fatalf("expected If-Modified-Since at Last-Modified to return 304 without a body, got %d body=%s", notModified.StatusCode, string(notModified.RawBody))
	}

	noneMatch, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "https://acctest.blob.core.windows.net/docs/properties.txt", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"If-None-Match": etag,
	}))
	if err != nil {
		t.Fatalf("none-match blob properties returned error: %v", err)
	}
	if noneMatch.StatusCode != http.StatusNotModified || len(noneMatch.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match on properties read to return 304 without a body, got %d body=%s", noneMatch.StatusCode, string(noneMatch.RawBody))
	}

	unmodifiedFailed, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "https://acctest.blob.core.windows.net/docs/properties.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"If-Unmodified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("unmodified-since blob properties returned error: %v", err)
	}
	if unmodifiedFailed.StatusCode != http.StatusPreconditionFailed || unmodifiedFailed.Headers["x-ms-error-code"] != "ConditionNotMet" {
		t.Fatalf("expected older If-Unmodified-Since on properties read to fail 412 ConditionNotMet, got %d headers=%v body=%s", unmodifiedFailed.StatusCode, unmodifiedFailed.Headers, string(unmodifiedFailed.RawBody))
	}
}

func TestBlobMetadataConditionalReadHonorsDateHeaders(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/metadata.txt", []byte("metadata data"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"x-ms-meta-owner": "platform",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}
	etag := putBlob.Headers["ETag"]
	if etag == "" {
		t.Fatal("expected uploaded blob ETag")
	}
	lastModified, err := time.Parse(http.TimeFormat, putBlob.Headers["Last-Modified"])
	if err != nil {
		t.Fatalf("expected parseable Last-Modified header, got %q: %v", putBlob.Headers["Last-Modified"], err)
	}
	beforeLastModified := lastModified.Add(-time.Minute).UTC().Format(http.TimeFormat)
	atLastModified := lastModified.UTC().Format(http.TimeFormat)

	metadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/metadata.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("conditional blob metadata returned error: %v", err)
	}
	if metadata.StatusCode != http.StatusOK || len(metadata.RawBody) != 0 || metadata.Headers["x-ms-meta-owner"] != "platform" {
		t.Fatalf("expected older If-Modified-Since to return metadata headers without a body, got status=%d headers=%v body=%s", metadata.StatusCode, metadata.Headers, string(metadata.RawBody))
	}

	notModified, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/metadata.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"If-Modified-Since": atLastModified,
	}))
	if err != nil {
		t.Fatalf("not-modified blob metadata returned error: %v", err)
	}
	if notModified.StatusCode != http.StatusNotModified || len(notModified.RawBody) != 0 {
		t.Fatalf("expected If-Modified-Since at Last-Modified to return 304 without a body, got %d body=%s", notModified.StatusCode, string(notModified.RawBody))
	}

	noneMatch, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/metadata.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"If-None-Match": etag,
	}))
	if err != nil {
		t.Fatalf("none-match blob metadata returned error: %v", err)
	}
	if noneMatch.StatusCode != http.StatusNotModified || len(noneMatch.RawBody) != 0 {
		t.Fatalf("expected matching If-None-Match on metadata read to return 304 without a body, got %d body=%s", noneMatch.StatusCode, string(noneMatch.RawBody))
	}

	unmodifiedFailed, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/metadata.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"If-Unmodified-Since": beforeLastModified,
	}))
	if err != nil {
		t.Fatalf("unmodified-since blob metadata returned error: %v", err)
	}
	if unmodifiedFailed.StatusCode != http.StatusPreconditionFailed || unmodifiedFailed.Headers["x-ms-error-code"] != "ConditionNotMet" {
		t.Fatalf("expected older If-Unmodified-Since on metadata read to fail 412 ConditionNotMet, got %d headers=%v body=%s", unmodifiedFailed.StatusCode, unmodifiedFailed.Headers, string(unmodifiedFailed.RawBody))
	}
}

func TestBlobDataPlaneSetAndGetMetadata(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("metadata body"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"x-ms-meta-owner": "initial",
		"Content-Type":    "text/plain",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}
	oldETag := putBlob.Headers["ETag"]

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":       "2023-11-03",
		"x-ms-meta-owner":    "updated",
		"x-ms-meta-purpose":  "blob-parity",
		"x-ms-meta-priority": "high",
	}))
	if err != nil {
		t.Fatalf("set blob metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected set metadata status 200, got %d; body=%s", setMetadata.StatusCode, string(setMetadata.RawBody))
	}
	if setMetadata.Headers["ETag"] == "" || setMetadata.Headers["ETag"] == oldETag {
		t.Fatalf("expected metadata update to return a changed ETag, old=%q new=%q", oldETag, setMetadata.Headers["ETag"])
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob metadata returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get metadata status 200, got %d; body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if getMetadata.Headers["x-ms-meta-owner"] != "updated" ||
		getMetadata.Headers["x-ms-meta-purpose"] != "blob-parity" ||
		getMetadata.Headers["x-ms-meta-priority"] != "high" ||
		getMetadata.Headers["x-ms-meta-missing"] != "" {
		t.Fatalf("unexpected metadata headers: %v", getMetadata.Headers)
	}
	if len(getMetadata.RawBody) != 0 {
		t.Fatalf("expected metadata GET to return no blob body, got %q", string(getMetadata.RawBody))
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get blob after metadata update returned error: %v", err)
	}
	if string(getBlob.RawBody) != "metadata body" || getBlob.RawContentType != "text/plain" {
		t.Fatalf("expected metadata update to preserve blob content and content type, got body=%q contentType=%q", string(getBlob.RawBody), getBlob.RawContentType)
	}
	if getBlob.Headers["x-ms-meta-owner"] != "updated" || getBlob.Headers["x-ms-meta-purpose"] != "blob-parity" {
		t.Fatalf("expected normal GET to expose updated metadata, got %v", getBlob.Headers)
	}
}

func TestBlobDataPlaneHeadSnapshotMetadataReturnsPointInTimeHeaders(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs/snapshot-metadata.txt"

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, []byte("original metadata"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"x-ms-meta-owner": "snapshot",
		"x-ms-meta-stage": "initial",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d body=%s", putBlob.StatusCode, string(putBlob.RawBody))
	}

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot blob returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot response to include x-ms-snapshot, got headers=%v", snapshotResp.Headers)
	}

	updateBaseMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-meta-owner":        "base",
		"x-ms-meta-stage":        "updated",
		"x-ms-meta-current":      "true",
		"x-ms-client-request-id": "metadata-head-snapshot",
	}))
	if err != nil {
		t.Fatalf("update base metadata returned error: %v", err)
	}
	if updateBaseMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected update base metadata status 200, got %d body=%s", updateBaseMetadata.StatusCode, string(updateBaseMetadata.RawBody))
	}

	headSnapshotMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?comp=metadata&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head snapshot metadata returned error: %v", err)
	}
	if headSnapshotMetadata.StatusCode != http.StatusOK ||
		len(headSnapshotMetadata.RawBody) != 0 ||
		headSnapshotMetadata.Headers["x-ms-meta-owner"] != "snapshot" ||
		headSnapshotMetadata.Headers["x-ms-meta-stage"] != "initial" ||
		headSnapshotMetadata.Headers["x-ms-meta-current"] != "" {
		t.Fatalf("expected HEAD snapshot metadata to return original metadata headers without a body, got status=%d headers=%v body=%s", headSnapshotMetadata.StatusCode, headSnapshotMetadata.Headers, string(headSnapshotMetadata.RawBody))
	}
}

func TestBlobContainerDataPlaneSetAndGetMetadata(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	metadataURL := containerURL + "&comp=metadata"

	createContainer, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, map[string]string{
		"x-ms-meta-team": "platform",
	}))
	if err != nil {
		t.Fatalf("create container returned error: %v", err)
	}
	if createContainer.StatusCode != http.StatusCreated {
		t.Fatalf("expected create container status 201, got %d; body=%s", createContainer.StatusCode, string(createContainer.RawBody))
	}
	oldETag := createContainer.Headers["ETag"]

	getInitialMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, metadataURL, nil, nil))
	if err != nil {
		t.Fatalf("get initial container metadata returned error: %v", err)
	}
	if getInitialMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected initial container metadata status 200, got %d; body=%s", getInitialMetadata.StatusCode, string(getInitialMetadata.RawBody))
	}
	if getInitialMetadata.Headers["x-ms-meta-team"] != "platform" || len(getInitialMetadata.RawBody) != 0 {
		t.Fatalf("expected initial metadata headers without body, headers=%v body=%q", getInitialMetadata.Headers, string(getInitialMetadata.RawBody))
	}

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, metadataURL, nil, map[string]string{
		"x-ms-meta-owner":   "docs",
		"x-ms-meta-purpose": "container-parity",
	}))
	if err != nil {
		t.Fatalf("set container metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected set container metadata status 200, got %d; body=%s", setMetadata.StatusCode, string(setMetadata.RawBody))
	}
	if setMetadata.Headers["ETag"] == "" || setMetadata.Headers["ETag"] == oldETag {
		t.Fatalf("expected changed container ETag, old=%q new=%q", oldETag, setMetadata.Headers["ETag"])
	}

	getUpdatedMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, metadataURL, nil, nil))
	if err != nil {
		t.Fatalf("get updated container metadata returned error: %v", err)
	}
	if getUpdatedMetadata.Headers["x-ms-meta-owner"] != "docs" ||
		getUpdatedMetadata.Headers["x-ms-meta-purpose"] != "container-parity" ||
		getUpdatedMetadata.Headers["x-ms-meta-team"] != "" ||
		len(getUpdatedMetadata.RawBody) != 0 {
		t.Fatalf("expected replaced container metadata without body, headers=%v body=%q", getUpdatedMetadata.Headers, string(getUpdatedMetadata.RawBody))
	}
}

func TestBlobContainerDataPlaneGetProperties(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"

	createContainer, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, map[string]string{
		"x-ms-meta-team": "platform",
	}))
	if err != nil {
		t.Fatalf("create container returned error: %v", err)
	}
	if createContainer.StatusCode != http.StatusCreated {
		t.Fatalf("expected create container status 201, got %d; body=%s", createContainer.StatusCode, string(createContainer.RawBody))
	}

	getProperties, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("get container properties returned error: %v", err)
	}
	if getProperties.StatusCode != http.StatusOK {
		t.Fatalf("expected container properties status 200, got %d; body=%s", getProperties.StatusCode, string(getProperties.RawBody))
	}
	if getProperties.Headers["ETag"] == "" ||
		getProperties.Headers["Last-Modified"] == "" ||
		getProperties.Headers["x-ms-version"] == "" ||
		getProperties.Headers["x-ms-meta-team"] != "platform" ||
		len(getProperties.RawBody) != 0 {
		t.Fatalf("expected container properties headers without body, headers=%v body=%q", getProperties.Headers, string(getProperties.RawBody))
	}
}

func TestBlobContainerDataPlaneLeaseAcquireReleaseAndDeleteEnforcement(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, map[string]string{
		"x-ms-meta-team": "platform",
	}))

	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("head container before lease returned error: %v", err)
	}
	leaseID := "11111111-1111-1111-1111-111111111111"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("acquire container lease returned error: %v", err)
	}
	if acquire.StatusCode != http.StatusCreated ||
		acquire.Headers["x-ms-lease-id"] != leaseID ||
		acquire.Headers["ETag"] != headBefore.Headers["ETag"] ||
		acquire.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected acquire to return lease ID without changing container properties, status=%d headers=%v before=%v", acquire.StatusCode, acquire.Headers, headBefore.Headers)
	}

	headLeased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("head leased container returned error: %v", err)
	}
	if headLeased.Headers["x-ms-lease-status"] != "locked" ||
		headLeased.Headers["x-ms-lease-state"] != "leased" ||
		headLeased.Headers["x-ms-lease-duration"] != "infinite" {
		t.Fatalf("expected locked infinite lease headers, got %v", headLeased.Headers)
	}

	headWithWrongLease, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL, nil, map[string]string{
		"x-ms-lease-id": "22222222-2222-2222-2222-222222222222",
	}))
	if err != nil {
		t.Fatalf("head container with wrong lease returned error: %v", err)
	}
	if headWithWrongLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected container properties with wrong lease to fail with 412, got %d body=%s", headWithWrongLease.StatusCode, string(headWithWrongLease.RawBody))
	}

	deleteWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("delete without lease returned error: %v", err)
	}
	if deleteWithoutLease.StatusCode != http.StatusConflict {
		t.Fatalf("expected container delete without lease to fail with 409, got %d body=%s", deleteWithoutLease.StatusCode, string(deleteWithoutLease.RawBody))
	}

	release, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action": "release",
		"x-ms-lease-id":     leaseID,
	}))
	if err != nil {
		t.Fatalf("release container lease returned error: %v", err)
	}
	if release.StatusCode != http.StatusOK || release.Headers["ETag"] != headBefore.Headers["ETag"] {
		t.Fatalf("expected release status 200 with unchanged ETag, got status=%d headers=%v before=%v", release.StatusCode, release.Headers, headBefore.Headers)
	}

	deleteAfterRelease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("delete after release returned error: %v", err)
	}
	if deleteAfterRelease.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete after release to succeed with 202, got %d body=%s", deleteAfterRelease.StatusCode, string(deleteAfterRelease.RawBody))
	}
}

func TestBlobContainerDataPlaneLeaseRenewChangeAndBreak(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, nil))

	leaseA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	leaseB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	leaseC := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseA,
	}))
	if err != nil {
		t.Fatalf("acquire container lease returned error: %v", err)
	}

	renew, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action": "renew",
		"x-ms-lease-id":     leaseA,
	}))
	if err != nil {
		t.Fatalf("renew container lease returned error: %v", err)
	}
	if renew.StatusCode != http.StatusOK || renew.Headers["x-ms-lease-id"] != leaseA || renew.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected renew status 200 with current lease and unchanged ETag, got status=%d headers=%v acquire=%v", renew.StatusCode, renew.Headers, acquire.Headers)
	}

	change, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "change",
		"x-ms-lease-id":          leaseA,
		"x-ms-proposed-lease-id": leaseB,
	}))
	if err != nil {
		t.Fatalf("change container lease returned error: %v", err)
	}
	if change.StatusCode != http.StatusOK || change.Headers["x-ms-lease-id"] != leaseB || change.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected change status 200 with proposed lease and unchanged ETag, got status=%d headers=%v acquire=%v", change.StatusCode, change.Headers, acquire.Headers)
	}

	deleteWithOldLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, containerURL, nil, map[string]string{
		"x-ms-lease-id": leaseA,
	}))
	if err != nil {
		t.Fatalf("delete with old lease returned error: %v", err)
	}
	if deleteWithOldLease.StatusCode != http.StatusConflict {
		t.Fatalf("expected old lease delete to fail with 409, got %d body=%s", deleteWithOldLease.StatusCode, string(deleteWithOldLease.RawBody))
	}

	breakLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":       "break",
		"x-ms-lease-break-period": "0",
	}))
	if err != nil {
		t.Fatalf("break container lease returned error: %v", err)
	}
	if breakLease.StatusCode != http.StatusAccepted || breakLease.Headers["x-ms-lease-time"] != "0" || breakLease.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected break status 202 with lease time 0 and unchanged ETag, got status=%d headers=%v acquire=%v", breakLease.StatusCode, breakLease.Headers, acquire.Headers)
	}

	headBroken, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("head broken container lease returned error: %v", err)
	}
	if headBroken.Headers["x-ms-lease-status"] != "unlocked" || headBroken.Headers["x-ms-lease-state"] != "broken" {
		t.Fatalf("expected broken unlocked container lease headers, got %v", headBroken.Headers)
	}

	acquireAfterBreak, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseC,
	}))
	if err != nil {
		t.Fatalf("acquire after break returned error: %v", err)
	}
	if acquireAfterBreak.StatusCode != http.StatusCreated || acquireAfterBreak.Headers["x-ms-lease-id"] != leaseC {
		t.Fatalf("expected acquire after break to succeed with new lease, got status=%d headers=%v", acquireAfterBreak.StatusCode, acquireAfterBreak.Headers)
	}
}

func TestBlobContainerDataPlaneACLSetGetHeadAndReplace(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, nil))

	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL, nil, nil))
	if err != nil {
		t.Fatalf("head container before ACL returned error: %v", err)
	}
	firstACL := []byte(`<?xml version="1.0" encoding="utf-8"?>
<SignedIdentifiers>
  <SignedIdentifier>
    <Id>readers</Id>
    <AccessPolicy>
      <Start>2026-06-17T00:00:00.0000000Z</Start>
      <Expiry>2026-06-18T00:00:00.0000000Z</Expiry>
      <Permission>rl</Permission>
    </AccessPolicy>
  </SignedIdentifier>
  <SignedIdentifier>
    <Id>writers</Id>
    <AccessPolicy>
      <Start>2026-06-17T00:00:00.0000000Z</Start>
      <Expiry>2026-06-19T00:00:00.0000000Z</Expiry>
      <Permission>racwd</Permission>
    </AccessPolicy>
  </SignedIdentifier>
</SignedIdentifiers>`)
	setACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", firstACL, map[string]string{
		"x-ms-blob-public-access": "container",
	}))
	if err != nil {
		t.Fatalf("set container ACL returned error: %v", err)
	}
	if setACL.StatusCode != http.StatusOK || setACL.Headers["ETag"] == "" || setACL.Headers["ETag"] == headBefore.Headers["ETag"] || setACL.Headers["Last-Modified"] == "" {
		t.Fatalf("expected set ACL to update container headers, status=%d headers=%v before=%v", setACL.StatusCode, setACL.Headers, headBefore.Headers)
	}

	getACL, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, nil))
	if err != nil {
		t.Fatalf("get container ACL returned error: %v", err)
	}
	if getACL.StatusCode != http.StatusOK || getACL.RawContentType != "application/xml" {
		t.Fatalf("expected get ACL status 200 XML, got status=%d contentType=%q body=%s", getACL.StatusCode, getACL.RawContentType, string(getACL.RawBody))
	}
	if getACL.Headers["x-ms-blob-public-access"] != "container" || getACL.Headers["ETag"] != setACL.Headers["ETag"] || getACL.Headers["Last-Modified"] != setACL.Headers["Last-Modified"] {
		t.Fatalf("expected get ACL to return public access and current headers, get=%v set=%v", getACL.Headers, setACL.Headers)
	}
	body := string(getACL.RawBody)
	for _, fragment := range []string{
		"<SignedIdentifiers>",
		"<Id>readers</Id>",
		"<Permission>rl</Permission>",
		"<Id>writers</Id>",
		"<Permission>racwd</Permission>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected container ACL body to include %q, got %s", fragment, body)
		}
	}

	headACL, err := svc.HandleRequest(storageCtx(t, http.MethodHead, containerURL+"&comp=acl", nil, nil))
	if err != nil {
		t.Fatalf("head container ACL returned error: %v", err)
	}
	if headACL.StatusCode != http.StatusOK || len(headACL.RawBody) != 0 || headACL.Headers["x-ms-blob-public-access"] != "container" || headACL.Headers["ETag"] != setACL.Headers["ETag"] {
		t.Fatalf("expected HEAD container ACL headers without body, status=%d headers=%v body=%q", headACL.StatusCode, headACL.Headers, string(headACL.RawBody))
	}

	replacementACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>private-read</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)
	replaceACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", replacementACL, nil))
	if err != nil {
		t.Fatalf("replace container ACL returned error: %v", err)
	}
	if replaceACL.StatusCode != http.StatusOK || replaceACL.Headers["ETag"] == setACL.Headers["ETag"] {
		t.Fatalf("expected replacing ACL to rotate ETag, status=%d headers=%v previous=%v", replaceACL.StatusCode, replaceACL.Headers, setACL.Headers)
	}
	replaced, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, nil))
	if err != nil {
		t.Fatalf("get replaced container ACL returned error: %v", err)
	}
	replacedBody := string(replaced.RawBody)
	if replaced.Headers["x-ms-blob-public-access"] != "" || strings.Contains(replacedBody, "<Id>readers</Id>") || !strings.Contains(replacedBody, "<Id>private-read</Id>") {
		t.Fatalf("expected container ACL replacement semantics and cleared public access, headers=%v body=%s", replaced.Headers, replacedBody)
	}
}

func TestBlobContainerDataPlaneACLValidatesLeaseHeaders(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, nil))
	validACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>policy</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)

	setWithAbsentLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, map[string]string{
		"x-ms-lease-id": "absent-lease",
	}))
	if err != nil {
		t.Fatalf("set ACL with absent lease returned error: %v", err)
	}
	if setWithAbsentLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set ACL with lease ID on available container to fail 412, got %d body=%s", setWithAbsentLease.StatusCode, string(setWithAbsentLease.RawBody))
	}

	leaseID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))

	setWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, nil))
	if err != nil {
		t.Fatalf("set ACL without active lease ID returned error: %v", err)
	}
	if setWithoutLease.StatusCode != http.StatusOK {
		t.Fatalf("expected set ACL without optional lease ID to succeed, got %d body=%s", setWithoutLease.StatusCode, string(setWithoutLease.RawBody))
	}

	setWithWrongLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, map[string]string{
		"x-ms-lease-id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}))
	if err != nil {
		t.Fatalf("set ACL with wrong lease returned error: %v", err)
	}
	if setWithWrongLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set ACL with wrong active lease ID to fail 412, got %d body=%s", setWithWrongLease.StatusCode, string(setWithWrongLease.RawBody))
	}

	setWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, map[string]string{
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("set ACL with active lease returned error: %v", err)
	}
	if setWithLease.StatusCode != http.StatusOK {
		t.Fatalf("expected set ACL with active lease to succeed, got %d body=%s", setWithLease.StatusCode, string(setWithLease.RawBody))
	}

	getWrongLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, map[string]string{
		"x-ms-lease-id": "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
	}))
	if err != nil {
		t.Fatalf("get ACL with wrong lease returned error: %v", err)
	}
	if getWrongLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected get ACL with wrong lease to fail 412, got %d body=%s", getWrongLease.StatusCode, string(getWrongLease.RawBody))
	}

	getWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, nil))
	if err != nil {
		t.Fatalf("get ACL without optional lease ID returned error: %v", err)
	}
	if getWithoutLease.StatusCode != http.StatusOK || !strings.Contains(string(getWithoutLease.RawBody), "<Id>policy</Id>") {
		t.Fatalf("expected get ACL without optional lease ID to return stored policy, status=%d body=%s", getWithoutLease.StatusCode, string(getWithoutLease.RawBody))
	}

	getWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, map[string]string{
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("get ACL with active lease returned error: %v", err)
	}
	if getWithLease.StatusCode != http.StatusOK || !strings.Contains(string(getWithLease.RawBody), "<Id>policy</Id>") {
		t.Fatalf("expected get ACL with active lease to return stored policy, status=%d body=%s", getWithLease.StatusCode, string(getWithLease.RawBody))
	}
}

func TestBlobContainerDataPlaneACLRejectsInvalidUpdates(t *testing.T) {
	svc := storage.New()
	containerURL := "http://localhost:4577/devstoreaccount1/docs?restype=container"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL, nil, nil))
	validACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>policy</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, map[string]string{
		"x-ms-blob-public-access": "blob",
	}))
	if err != nil {
		t.Fatalf("set valid ACL returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusOK {
		t.Fatalf("expected valid ACL status 200, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	tooManyACL := []byte(`<SignedIdentifiers>
<SignedIdentifier><Id>p1</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p2</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p3</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p4</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p5</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p6</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
</SignedIdentifiers>`)
	setTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", tooManyACL, map[string]string{
		"x-ms-blob-public-access": "container",
	}))
	if err != nil {
		t.Fatalf("set too many container ACL policies returned error: %v", err)
	}
	if setTooMany.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many policies to fail 400, got %d body=%s", setTooMany.StatusCode, string(setTooMany.RawBody))
	}

	longIDACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>abcdefghijklmnopqrstuvwxyzabcdefghijklmnopqrstuvwxyzabcdefghijklm</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)
	setLongID, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", longIDACL, nil))
	if err != nil {
		t.Fatalf("set long ID ACL returned error: %v", err)
	}
	if setLongID.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected long policy ID to fail 400, got %d body=%s", setLongID.StatusCode, string(setLongID.RawBody))
	}

	setInvalidPublicAccess, err := svc.HandleRequest(storageCtx(t, http.MethodPut, containerURL+"&comp=acl", validACL, map[string]string{
		"x-ms-blob-public-access": "account",
	}))
	if err != nil {
		t.Fatalf("set invalid public access ACL returned error: %v", err)
	}
	if setInvalidPublicAccess.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid public access to fail 400, got %d body=%s", setInvalidPublicAccess.StatusCode, string(setInvalidPublicAccess.RawBody))
	}

	getAfterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, containerURL+"&comp=acl", nil, nil))
	if err != nil {
		t.Fatalf("get ACL after invalid updates returned error: %v", err)
	}
	if getAfterInvalid.Headers["x-ms-blob-public-access"] != "blob" || !strings.Contains(string(getAfterInvalid.RawBody), "<Id>policy</Id>") {
		t.Fatalf("expected invalid ACL updates not to replace existing state, headers=%v body=%s", getAfterInvalid.Headers, string(getAfterInvalid.RawBody))
	}
}

func TestBlobDataPlaneListContainersCanIncludeMetadataAndPage(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/beta?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/alpha?restype=container", nil, map[string]string{
		"x-ms-meta-owner": "sdk",
	}))

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata&maxresults=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("first list containers page returned error: %v", err)
	}
	if firstPage.StatusCode != http.StatusOK {
		t.Fatalf("expected first list containers status 200, got %d; body=%s", firstPage.StatusCode, string(firstPage.RawBody))
	}
	firstBody := string(firstPage.RawBody)
	if !strings.Contains(firstBody, `<Name>alpha</Name>`) || strings.Contains(firstBody, `<Name>beta</Name>`) {
		t.Fatalf("unexpected first list containers page: %s", firstBody)
	}
	if !strings.Contains(firstBody, `<Metadata>`) ||
		!strings.Contains(firstBody, `<owner>sdk</owner>`) ||
		!strings.Contains(firstBody, `<NextMarker>beta</NextMarker>`) ||
		!strings.Contains(firstBody, `ServiceEndpoint="https://devstoreaccount1.blob.core.windows.net/"`) {
		t.Fatalf("expected metadata, endpoint, and continuation marker in first page, got: %s", firstBody)
	}

	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata&maxresults=1&marker=beta", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("second list containers page returned error: %v", err)
	}
	secondBody := string(secondPage.RawBody)
	if strings.Contains(secondBody, `<Name>alpha</Name>`) || !strings.Contains(secondBody, `<Name>beta</Name>`) {
		t.Fatalf("unexpected second list containers page: %s", secondBody)
	}
	if strings.Contains(secondBody, `<NextMarker>`) {
		t.Fatalf("expected final list containers page without NextMarker, got: %s", secondBody)
	}
}

func TestBlobListContainersRejectsInvalidMaxResults(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/docs?restype=container", nil, nil))

	listContainers, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&maxresults=0", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list containers returned error: %v", err)
	}
	if listContainers.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid container maxresults status 400, got %d; body=%s", listContainers.StatusCode, string(listContainers.RawBody))
	}
}

func TestBlobListCanIncludeMetadata(t *testing.T) {
	svc := storage.New()

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("metadata data"), map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-blob-type":  "BlockBlob",
		"x-ms-meta-owner": "compat",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d", putBlob.StatusCode)
	}

	listBlobs, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs?restype=container&comp=list&include=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list blobs with metadata returned error: %v", err)
	}
	if listBlobs.StatusCode != http.StatusOK {
		t.Fatalf("expected list blobs status 200, got %d; body=%s", listBlobs.StatusCode, string(listBlobs.RawBody))
	}
	body := string(listBlobs.RawBody)
	if !strings.Contains(body, "<Metadata>") || !strings.Contains(body, "<owner>compat</owner>") {
		t.Fatalf("expected blob metadata in listing XML, got %s", body)
	}
}

func TestTableDataPlaneLifecycleFilteringAndETags(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.table.core.windows.net/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	createdTable := decodeJSONBody(t, createTable)
	if createdTable["TableName"] != "Tasks" {
		t.Fatalf("unexpected created table: %v", createdTable)
	}

	duplicateTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.table.core.windows.net/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("duplicate table returned error: %v", err)
	}
	if duplicateTable.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate table status 409, got %d; body=%s", duplicateTable.StatusCode, string(duplicateTable.RawBody))
	}

	listTables, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.table.core.windows.net/Tables", nil, headers))
	if err != nil {
		t.Fatalf("list tables returned error: %v", err)
	}
	listedTables := decodeJSONBody(t, listTables)
	if len(listedTables["value"].([]any)) != 1 {
		t.Fatalf("expected one table, got %v", listedTables)
	}

	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.table.core.windows.net/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"hello","Score":50}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	inserted := decodeJSONBody(t, insertEntity)
	if inserted["PartitionKey"] != "p1" || inserted["RowKey"] != "r1" || inserted["Value"] != "hello" {
		t.Fatalf("unexpected inserted entity: %v", inserted)
	}
	oldETag := insertEntity.Headers["ETag"]
	if oldETag == "" {
		t.Fatalf("expected entity ETag")
	}

	getEntityURL := "https://acctest.table.core.windows.net/Tasks(PartitionKey='p1',RowKey='r1')"
	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, getEntityURL, nil, headers))
	if err != nil {
		t.Fatalf("get entity returned error: %v", err)
	}
	got := decodeJSONBody(t, getEntity)
	if got["Value"] != "hello" || got["odata.etag"] == "" || getEntity.Headers["ETag"] == "" {
		t.Fatalf("unexpected fetched entity: %v headers=%v", got, getEntity.Headers)
	}

	updateReqHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     oldETag,
	}
	updateEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPut, getEntityURL, []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"updated","Score":80}`), updateReqHeaders))
	if err != nil {
		t.Fatalf("update entity returned error: %v", err)
	}
	if updateEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected update entity status 204, got %d; body=%s", updateEntity.StatusCode, string(updateEntity.RawBody))
	}
	newETag := updateEntity.Headers["ETag"]
	if newETag == "" || newETag == oldETag {
		t.Fatalf("expected changed ETag, old=%q new=%q", oldETag, newETag)
	}

	staleUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, getEntityURL, []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"stale"}`), updateReqHeaders))
	if err != nil {
		t.Fatalf("stale update returned error: %v", err)
	}
	if staleUpdate.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stale update status 412, got %d; body=%s", staleUpdate.StatusCode, string(staleUpdate.RawBody))
	}

	_, err = svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.table.core.windows.net/Tasks", []byte(`{"PartitionKey":"p2","RowKey":"r1","Value":"other","Score":10}`), headers))
	if err != nil {
		t.Fatalf("insert second entity returned error: %v", err)
	}
	queryURL := "https://acctest.table.core.windows.net/Tasks()?$filter=PartitionKey%20eq%20'p1'&$select=Value,Score"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("query entities returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one filtered entity, got %v", query)
	}
	projected := values[0].(map[string]any)
	if projected["PartitionKey"] != "p1" || projected["RowKey"] != "r1" || projected["Value"] != "updated" || projected["Score"] != float64(80) {
		t.Fatalf("unexpected projected entity: %v", projected)
	}

	deleteHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"If-Match":     newETag,
	}
	deleteEntity, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, getEntityURL, nil, deleteHeaders))
	if err != nil {
		t.Fatalf("delete entity returned error: %v", err)
	}
	if deleteEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete entity status 204, got %d; body=%s", deleteEntity.StatusCode, string(deleteEntity.RawBody))
	}

	missingEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, getEntityURL, nil, headers))
	if err != nil {
		t.Fatalf("get missing entity returned error: %v", err)
	}
	if missingEntity.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing entity status 404, got %d; body=%s", missingEntity.StatusCode, string(missingEntity.RawBody))
	}

	deleteTable, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://acctest.table.core.windows.net/Tables('Tasks')", nil, headers))
	if err != nil {
		t.Fatalf("delete table returned error: %v", err)
	}
	if deleteTable.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete table status 204, got %d; body=%s", deleteTable.StatusCode, string(deleteTable.RawBody))
	}
}

func TestFileShareDataPlaneLifecycle(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	headers := map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-meta-owner":        "platform",
		"x-ms-share-quota":       "64",
		"x-ms-access-tier":       "Hot",
		"x-ms-enabled-protocols": "SMB",
	}

	createShare, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, headers))
	if err != nil {
		t.Fatalf("create share returned error: %v", err)
	}
	if createShare.StatusCode != http.StatusCreated {
		t.Fatalf("expected create share status 201, got %d; body=%s", createShare.StatusCode, string(createShare.RawBody))
	}
	if createShare.Headers["ETag"] == "" || createShare.Headers["Last-Modified"] == "" || createShare.Headers["x-ms-version"] == "" {
		t.Fatalf("expected create share Azure Files headers, got %v", createShare.Headers)
	}

	duplicateShare, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, headers))
	if err != nil {
		t.Fatalf("duplicate share returned error: %v", err)
	}
	if duplicateShare.StatusCode != http.StatusConflict {
		t.Fatalf("expected duplicate share status 409, got %d; body=%s", duplicateShare.StatusCode, string(duplicateShare.RawBody))
	}

	getShare, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share properties returned error: %v", err)
	}
	if getShare.StatusCode != http.StatusOK {
		t.Fatalf("expected get share properties status 200, got %d; body=%s", getShare.StatusCode, string(getShare.RawBody))
	}
	if getShare.Headers["x-ms-meta-owner"] != "platform" || getShare.Headers["x-ms-share-quota"] != "64" || getShare.Headers["x-ms-access-tier"] != "Hot" || getShare.Headers["x-ms-enabled-protocols"] != "SMB" {
		t.Fatalf("expected share properties and metadata headers, got %v", getShare.Headers)
	}
	if len(getShare.RawBody) != 0 {
		t.Fatalf("expected HEAD share properties to return no body, got %q", string(getShare.RawBody))
	}

	listShares, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1-file?comp=list&include=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list shares returned error: %v", err)
	}
	if listShares.StatusCode != http.StatusOK {
		t.Fatalf("expected list shares status 200, got %d; body=%s", listShares.StatusCode, string(listShares.RawBody))
	}
	body := string(listShares.RawBody)
	for _, want := range []string{
		`ServiceEndpoint="https://devstoreaccount1.file.core.windows.net/"`,
		"<Name>reports</Name>",
		"<Quota>64</Quota>",
		"<AccessTier>Hot</AccessTier>",
		"<EnabledProtocols>SMB</EnabledProtocols>",
		"<owner>platform</owner>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected list shares body to contain %s, got: %s", want, body)
		}
	}

	deleteShare, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete share returned error: %v", err)
	}
	if deleteShare.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete share status 202, got %d; body=%s", deleteShare.StatusCode, string(deleteShare.RawBody))
	}
}

func TestFileDataPlaneDirectoryFileRangeAndList(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	createDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "ops",
	}))
	if err != nil {
		t.Fatalf("create directory returned error: %v", err)
	}
	if createDirectory.StatusCode != http.StatusCreated {
		t.Fatalf("expected create directory status 201, got %d; body=%s", createDirectory.StatusCode, string(createDirectory.RawBody))
	}
	if createDirectory.Headers["ETag"] == "" || createDirectory.Headers["Last-Modified"] == "" {
		t.Fatalf("expected directory create headers, got %v", createDirectory.Headers)
	}

	createFile, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "11",
		"x-ms-content-type":   "text/plain",
		"x-ms-meta-source":    "range-test",
	}))
	if err != nil {
		t.Fatalf("create file returned error: %v", err)
	}
	if createFile.StatusCode != http.StatusCreated {
		t.Fatalf("expected create file status 201, got %d; body=%s", createFile.StatusCode, string(createFile.RawBody))
	}

	putRange, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs/today.txt?comp=range", []byte("hello azure"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-10",
	}))
	if err != nil {
		t.Fatalf("put range returned error: %v", err)
	}
	if putRange.StatusCode != http.StatusCreated {
		t.Fatalf("expected put range status 201, got %d; body=%s", putRange.StatusCode, string(putRange.RawBody))
	}
	if putRange.Headers["ETag"] == "" || putRange.Headers["Last-Modified"] == "" {
		t.Fatalf("expected put range to return file headers, got %v", putRange.Headers)
	}

	getFile, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get file returned error: %v", err)
	}
	if getFile.StatusCode != http.StatusOK {
		t.Fatalf("expected get file status 200, got %d; body=%s", getFile.StatusCode, string(getFile.RawBody))
	}
	if string(getFile.RawBody) != "hello azure" {
		t.Fatalf("expected full file contents, got %q", string(getFile.RawBody))
	}
	if getFile.Headers["Content-Length"] != "11" || getFile.Headers["Content-Type"] != "text/plain" || getFile.Headers["x-ms-meta-source"] != "range-test" || getFile.Headers["Accept-Ranges"] != "bytes" {
		t.Fatalf("expected file content headers, got %v", getFile.Headers)
	}

	getRange, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-range":   "bytes=6-10",
	}))
	if err != nil {
		t.Fatalf("get file range returned error: %v", err)
	}
	if getRange.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected get file range status 206, got %d; body=%s", getRange.StatusCode, string(getRange.RawBody))
	}
	if string(getRange.RawBody) != "azure" || getRange.Headers["Content-Range"] != "bytes 6-10/11" || getRange.Headers["Content-Length"] != "5" {
		t.Fatalf("expected partial file range response, body=%q headers=%v", string(getRange.RawBody), getRange.Headers)
	}

	listDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs?restype=directory&comp=list", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list directory returned error: %v", err)
	}
	if listDirectory.StatusCode != http.StatusOK {
		t.Fatalf("expected list directory status 200, got %d; body=%s", listDirectory.StatusCode, string(listDirectory.RawBody))
	}
	listBody := string(listDirectory.RawBody)
	for _, want := range []string{
		`ShareName="reports"`,
		`DirectoryPath="logs"`,
		"<File>",
		"<Name>today.txt</Name>",
		"<Content-Length>11</Content-Length>",
	} {
		if !strings.Contains(listBody, want) {
			t.Fatalf("expected directory listing to contain %s, got: %s", want, listBody)
		}
	}
}

func TestFileDataPlaneListRangesTracksUpdatesClearsAndFilters(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/disk.img", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "1024",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/disk.img?comp=range", bytes.Repeat([]byte("a"), 128), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-127",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/disk.img?comp=range", bytes.Repeat([]byte("b"), 256), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=512-767",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/disk.img?comp=range", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "clear",
		"x-ms-range":   "bytes=64-127",
	}))

	listRanges, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/disk.img?comp=rangelist", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list ranges returned error: %v", err)
	}
	if listRanges.StatusCode != http.StatusOK {
		t.Fatalf("expected list ranges status 200, got %d; body=%s", listRanges.StatusCode, string(listRanges.RawBody))
	}
	if listRanges.Headers["x-ms-content-length"] != "1024" || listRanges.Headers["ETag"] == "" || listRanges.RawContentType != "application/xml" {
		t.Fatalf("expected list ranges headers, got contentType=%q headers=%v", listRanges.RawContentType, listRanges.Headers)
	}
	body := string(listRanges.RawBody)
	for _, want := range []string{
		"<Ranges>",
		"<Start>0</Start>",
		"<End>63</End>",
		"<Start>512</Start>",
		"<End>767</End>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected range list to contain %s, got: %s", want, body)
		}
	}
	if strings.Contains(body, "<Start>64</Start>") || strings.Contains(body, "<End>127</End>") {
		t.Fatalf("expected cleared range to be absent, got: %s", body)
	}

	filtered, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/disk.img?comp=rangelist", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-range":   "bytes=600-900",
	}))
	if err != nil {
		t.Fatalf("filtered list ranges returned error: %v", err)
	}
	if filtered.StatusCode != http.StatusOK {
		t.Fatalf("expected filtered list ranges status 200, got %d; body=%s", filtered.StatusCode, string(filtered.RawBody))
	}
	filteredBody := string(filtered.RawBody)
	for _, want := range []string{
		"<Start>600</Start>",
		"<End>767</End>",
	} {
		if !strings.Contains(filteredBody, want) {
			t.Fatalf("expected filtered range list to contain %s, got: %s", want, filteredBody)
		}
	}
	if strings.Contains(filteredBody, "<Start>0</Start>") || strings.Contains(filteredBody, "<End>63</End>") {
		t.Fatalf("expected filtered range list to exclude first range, got: %s", filteredBody)
	}
}

func TestFileDataPlaneRenameFilePreservesContentMetadataAndRanges(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	sourceURL := baseURL + "/logs/today.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/archive?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "11",
		"x-ms-content-type":   "text/plain",
		"x-ms-meta-source":    "rename-test",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"?comp=range", []byte("hello azure"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-10",
	}))

	renameFile, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/archive/today.txt?comp=rename", nil, map[string]string{
		"x-ms-version":            "2023-11-03",
		"x-ms-file-rename-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("rename file returned error: %v", err)
	}
	if renameFile.StatusCode != http.StatusOK {
		t.Fatalf("expected rename file status 200, got %d; body=%s", renameFile.StatusCode, string(renameFile.RawBody))
	}
	if len(renameFile.RawBody) != 0 || renameFile.Headers["ETag"] == "" || renameFile.Headers["Last-Modified"] == "" {
		t.Fatalf("expected rename file headers without body, body=%q headers=%v", string(renameFile.RawBody), renameFile.Headers)
	}

	getRenamed, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/archive/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get renamed file returned error: %v", err)
	}
	if getRenamed.StatusCode != http.StatusOK || string(getRenamed.RawBody) != "hello azure" || getRenamed.Headers["x-ms-meta-source"] != "rename-test" || getRenamed.Headers["Content-Type"] != "text/plain" {
		t.Fatalf("expected renamed file content and properties, status=%d body=%q headers=%v", getRenamed.StatusCode, string(getRenamed.RawBody), getRenamed.Headers)
	}

	getSource, err := svc.HandleRequest(storageCtx(t, http.MethodGet, sourceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get source after rename returned error: %v", err)
	}
	if getSource.StatusCode != http.StatusNotFound {
		t.Fatalf("expected source file to be removed, got %d; body=%s", getSource.StatusCode, string(getSource.RawBody))
	}

	listRanges, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/archive/today.txt?comp=rangelist", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list ranges for renamed file returned error: %v", err)
	}
	rangeBody := string(listRanges.RawBody)
	if listRanges.StatusCode != http.StatusOK || !strings.Contains(rangeBody, "<Start>0</Start>") || !strings.Contains(rangeBody, "<End>10</End>") {
		t.Fatalf("expected renamed file ranges to be preserved, status=%d body=%s", listRanges.StatusCode, rangeBody)
	}
}

func TestFileDataPlaneRenameDirectoryMovesChildPaths(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	sourceURL := baseURL + "/logs"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"?restype=directory", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "ops",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"/today.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
		"x-ms-meta-source":    "directory-rename",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"/today.txt?comp=range", []byte("hello"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))

	renameDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/archive?restype=directory&comp=rename", nil, map[string]string{
		"x-ms-version":            "2023-11-03",
		"x-ms-file-rename-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("rename directory returned error: %v", err)
	}
	if renameDirectory.StatusCode != http.StatusOK {
		t.Fatalf("expected rename directory status 200, got %d; body=%s", renameDirectory.StatusCode, string(renameDirectory.RawBody))
	}
	if len(renameDirectory.RawBody) != 0 || renameDirectory.Headers["ETag"] == "" || renameDirectory.Headers["x-ms-meta-owner"] != "ops" {
		t.Fatalf("expected rename directory headers without body, body=%q headers=%v", string(renameDirectory.RawBody), renameDirectory.Headers)
	}

	headDestination, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/archive?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head destination directory returned error: %v", err)
	}
	if headDestination.StatusCode != http.StatusOK || headDestination.Headers["x-ms-meta-owner"] != "ops" {
		t.Fatalf("expected renamed directory properties, status=%d headers=%v", headDestination.StatusCode, headDestination.Headers)
	}

	getMovedFile, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/archive/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get moved file returned error: %v", err)
	}
	if getMovedFile.StatusCode != http.StatusOK || string(getMovedFile.RawBody) != "hello" || getMovedFile.Headers["x-ms-meta-source"] != "directory-rename" {
		t.Fatalf("expected moved child file content and metadata, status=%d body=%q headers=%v", getMovedFile.StatusCode, string(getMovedFile.RawBody), getMovedFile.Headers)
	}

	headSource, err := svc.HandleRequest(storageCtx(t, http.MethodHead, sourceURL+"?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head source directory returned error: %v", err)
	}
	if headSource.StatusCode != http.StatusNotFound {
		t.Fatalf("expected source directory to be removed, got %d; body=%s", headSource.StatusCode, string(headSource.RawBody))
	}

	getSourceChild, err := svc.HandleRequest(storageCtx(t, http.MethodGet, sourceURL+"/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get source child returned error: %v", err)
	}
	if getSourceChild.StatusCode != http.StatusNotFound {
		t.Fatalf("expected source child to be removed, got %d; body=%s", getSourceChild.StatusCode, string(getSourceChild.RawBody))
	}
}

func TestFileDataPlaneLeaseFileAcquireReleaseAndWriteEnforcement(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	fileURL := baseURL + "/logs/today.txt"
	leaseID := "11111111-1111-1111-1111-111111111111"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))
	headBefore, _ := svc.HandleRequest(storageCtx(t, http.MethodHead, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("acquire file lease returned error: %v", err)
	}
	if acquire.StatusCode != http.StatusCreated || acquire.Headers["x-ms-lease-id"] != leaseID || acquire.Headers["ETag"] != headBefore.Headers["ETag"] || acquire.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected acquire lease to return lease ID without file mutation, status=%d headers=%v before=%v", acquire.StatusCode, acquire.Headers, headBefore.Headers)
	}

	headLeased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head leased file returned error: %v", err)
	}
	if headLeased.Headers["x-ms-lease-status"] != "locked" || headLeased.Headers["x-ms-lease-state"] != "leased" {
		t.Fatalf("expected leased file property headers, got %v", headLeased.Headers)
	}

	writeWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("hello"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))
	if err != nil {
		t.Fatalf("write without lease returned error: %v", err)
	}
	if writeWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected write without lease to fail 412, got %d; body=%s", writeWithoutLease.StatusCode, string(writeWithoutLease.RawBody))
	}

	writeWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("hello"), map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-write":    "update",
		"x-ms-range":    "bytes=0-4",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("write with lease returned error: %v", err)
	}
	if writeWithLease.StatusCode != http.StatusCreated {
		t.Fatalf("expected write with lease to succeed, got %d; body=%s", writeWithLease.StatusCode, string(writeWithLease.RawBody))
	}

	deleteWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete without lease returned error: %v", err)
	}
	if deleteWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected delete without lease to fail 412, got %d; body=%s", deleteWithoutLease.StatusCode, string(deleteWithoutLease.RawBody))
	}

	release, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-lease-action": "release",
		"x-ms-lease-id":     leaseID,
	}))
	if err != nil {
		t.Fatalf("release file lease returned error: %v", err)
	}
	if release.StatusCode != http.StatusOK || release.Headers["ETag"] != writeWithLease.Headers["ETag"] {
		t.Fatalf("expected release status 200 with current ETag, got status=%d headers=%v write=%v", release.StatusCode, release.Headers, writeWithLease.Headers)
	}

	deleteAfterRelease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete after release returned error: %v", err)
	}
	if deleteAfterRelease.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete after release to succeed, got %d; body=%s", deleteAfterRelease.StatusCode, string(deleteAfterRelease.RawBody))
	}
}

func TestFileDataPlaneLeaseFileAvailableAndBrokenWriteOutcomes(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	fileURL := baseURL + "/logs/state.txt"
	leaseID := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))

	writeWithLeaseOnAvailable, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("hello"), map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-write":    "update",
		"x-ms-range":    "bytes=0-4",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("write with lease on available file returned error: %v", err)
	}
	if writeWithLeaseOnAvailable.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected write using lease ID on available file to fail 412, got %d; body=%s", writeWithLeaseOnAvailable.StatusCode, string(writeWithLeaseOnAvailable.RawBody))
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	breakLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":            "2023-11-03",
		"x-ms-lease-action":       "break",
		"x-ms-lease-break-period": "0",
	}))
	if err != nil {
		t.Fatalf("break file lease returned error: %v", err)
	}
	if breakLease.StatusCode != http.StatusAccepted || breakLease.Headers["x-ms-lease-time"] != "0" {
		t.Fatalf("expected break lease status 202 with zero remaining time, got status=%d headers=%v", breakLease.StatusCode, breakLease.Headers)
	}

	writeWithLeaseOnBroken, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("HELLO"), map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-write":    "update",
		"x-ms-range":    "bytes=0-4",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("write with lease on broken file returned error: %v", err)
	}
	if writeWithLeaseOnBroken.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected write using lease ID on broken file to fail 412, got %d; body=%s", writeWithLeaseOnBroken.StatusCode, string(writeWithLeaseOnBroken.RawBody))
	}

	writeWithoutLeaseOnBroken, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("HELLO"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))
	if err != nil {
		t.Fatalf("write without lease on broken file returned error: %v", err)
	}
	if writeWithoutLeaseOnBroken.StatusCode != http.StatusCreated {
		t.Fatalf("expected write without lease on broken file to succeed, got %d; body=%s", writeWithoutLeaseOnBroken.StatusCode, string(writeWithoutLeaseOnBroken.RawBody))
	}

	headAfterWrite, err := svc.HandleRequest(storageCtx(t, http.MethodHead, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head after broken lease write returned error: %v", err)
	}
	if headAfterWrite.Headers["x-ms-lease-status"] != "unlocked" || headAfterWrite.Headers["x-ms-lease-state"] != "available" {
		t.Fatalf("expected write without lease on broken file to make lease available, got %v", headAfterWrite.Headers)
	}
}

func TestFileDataPlaneLeaseFileReleaseBrokenLeaseWithMatchingID(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	fileURL := baseURL + "/logs/broken-release.txt"
	leaseID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "1",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":            "2023-11-03",
		"x-ms-lease-action":       "break",
		"x-ms-lease-break-period": "0",
	}))

	releaseBroken, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-lease-action": "release",
		"x-ms-lease-id":     leaseID,
	}))
	if err != nil {
		t.Fatalf("release broken file lease returned error: %v", err)
	}
	if releaseBroken.StatusCode != http.StatusOK {
		t.Fatalf("expected release with matching broken lease ID to succeed, got %d; body=%s", releaseBroken.StatusCode, string(releaseBroken.RawBody))
	}

	headReleased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head released broken lease returned error: %v", err)
	}
	if headReleased.Headers["x-ms-lease-status"] != "unlocked" || headReleased.Headers["x-ms-lease-state"] != "available" {
		t.Fatalf("expected released broken lease to become available, got %v", headReleased.Headers)
	}
}

func TestFileDataPlaneCopyFileCopiesContentPropertiesAndMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	sourceURL := baseURL + "/logs/source.txt"
	destinationURL := baseURL + "/archive/copied.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/archive?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "11",
		"x-ms-content-type":   "text/plain",
		"x-ms-content-md5":    "hM6zH9Y6z+XgKfS4Lx3pQw==",
		"x-ms-meta-source":    "file-copy",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"?comp=properties", nil, map[string]string{
		"x-ms-version":             "2023-11-03",
		"x-ms-content-type":        "text/plain",
		"x-ms-cache-control":       "max-age=60",
		"x-ms-content-encoding":    "gzip",
		"x-ms-content-language":    "en-US",
		"x-ms-content-disposition": "inline",
		"x-ms-content-md5":         "hM6zH9Y6z+XgKfS4Lx3pQw==",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"?comp=range", []byte("hello azure"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-10",
	}))

	copyResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("copy file returned error: %v", err)
	}
	if copyResp.StatusCode != http.StatusAccepted || copyResp.Headers["x-ms-copy-id"] == "" || copyResp.Headers["x-ms-copy-status"] != "success" {
		t.Fatalf("expected copy to return 202 with success copy headers, got status=%d headers=%v body=%s", copyResp.StatusCode, copyResp.Headers, string(copyResp.RawBody))
	}

	getDestination, err := svc.HandleRequest(storageCtx(t, http.MethodGet, destinationURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get copied file returned error: %v", err)
	}
	if getDestination.StatusCode != http.StatusOK || string(getDestination.RawBody) != "hello azure" {
		t.Fatalf("expected copied content, got status=%d body=%q", getDestination.StatusCode, string(getDestination.RawBody))
	}
	expectedHeaders := map[string]string{
		"x-ms-meta-source":    "file-copy",
		"Content-Type":        "text/plain",
		"Cache-Control":       "max-age=60",
		"Content-Encoding":    "gzip",
		"Content-Language":    "en-US",
		"Content-Disposition": "inline",
		"Content-MD5":         "hM6zH9Y6z+XgKfS4Lx3pQw==",
		"Content-Length":      "11",
	}
	for key, want := range expectedHeaders {
		if getDestination.Headers[key] != want {
			t.Fatalf("expected copied header %s=%q, got headers=%v", key, want, getDestination.Headers)
		}
	}
}

func TestFileDataPlaneCopyFileRequiresDestinationLeaseID(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	sourceURL := baseURL + "/source.txt"
	destinationURL := baseURL + "/destination.txt"
	leaseID := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "6",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL+"?comp=range", []byte("source"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-5",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "3",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL+"?comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))

	copyWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("copy without destination lease returned error: %v", err)
	}
	if copyWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected copy without destination lease to fail 412, got %d; body=%s", copyWithoutLease.StatusCode, string(copyWithoutLease.RawBody))
	}

	copyWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
		"x-ms-lease-id":    leaseID,
	}))
	if err != nil {
		t.Fatalf("copy with destination lease returned error: %v", err)
	}
	if copyWithLease.StatusCode != http.StatusAccepted || copyWithLease.Headers["x-ms-copy-status"] != "success" {
		t.Fatalf("expected copy with destination lease to succeed, got status=%d headers=%v body=%s", copyWithLease.StatusCode, copyWithLease.Headers, string(copyWithLease.RawBody))
	}

	headDestination, err := svc.HandleRequest(storageCtx(t, http.MethodHead, destinationURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head copied leased destination returned error: %v", err)
	}
	if headDestination.Headers["x-ms-lease-status"] != "locked" || headDestination.Headers["x-ms-lease-state"] != "leased" || headDestination.Headers["Content-Length"] != "6" {
		t.Fatalf("expected copied destination to keep lease and source length, got %v", headDestination.Headers)
	}
}

func TestFileDataPlaneSnapshotSharePreservesPointInTimeFilesAndListsSnapshots(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-file"
	baseURL := accountURL + "/reports"
	fileURL := baseURL + "/logs/today.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "base",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
		"x-ms-content-type":   "text/plain",
		"x-ms-meta-version":   "one",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("first"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot share returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshotResp.StatusCode != http.StatusCreated || snapshot == "" {
		t.Fatalf("expected snapshot share to return 201 with x-ms-snapshot, got status=%d headers=%v body=%s", snapshotResp.StatusCode, snapshotResp.Headers, string(snapshotResp.RawBody))
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=range", []byte("later"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))

	getBase, err := svc.HandleRequest(storageCtx(t, http.MethodGet, fileURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get base file returned error: %v", err)
	}
	if getBase.StatusCode != http.StatusOK || string(getBase.RawBody) != "later" {
		t.Fatalf("expected base file to reflect later write, got status=%d body=%q", getBase.StatusCode, string(getBase.RawBody))
	}

	getSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodGet, fileURL+"?sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot file returned error: %v", err)
	}
	if getSnapshot.StatusCode != http.StatusOK || string(getSnapshot.RawBody) != "first" || getSnapshot.Headers["x-ms-snapshot"] != snapshot || getSnapshot.Headers["x-ms-meta-version"] != "one" {
		t.Fatalf("expected snapshot file to return point-in-time content and headers, got status=%d body=%q headers=%v", getSnapshot.StatusCode, string(getSnapshot.RawBody), getSnapshot.Headers)
	}

	writeSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?sharesnapshot="+url.QueryEscape(snapshot)+"&comp=range", []byte("block"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))
	if err != nil {
		t.Fatalf("write snapshot file returned error: %v", err)
	}
	if writeSnapshot.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected snapshot writes to fail 400, got %d; body=%s", writeSnapshot.StatusCode, string(writeSnapshot.RawBody))
	}

	list, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata,snapshots", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list shares with snapshots returned error: %v", err)
	}
	body := string(list.RawBody)
	for _, fragment := range []string{
		"<Name>reports</Name>",
		"<Snapshot>" + snapshot + "</Snapshot>",
		"<Metadata><owner>base</owner></Metadata>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected list shares response to include %q, got %s", fragment, body)
		}
	}
}

func TestFileDataPlaneDeleteShareHonorsSnapshotOptions(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	snapshotResp, _ := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	snapshot := snapshotResp.Headers["x-ms-snapshot"]

	deleteBase, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete base share returned error: %v", err)
	}
	if deleteBase.StatusCode != http.StatusConflict {
		t.Fatalf("expected base share delete with snapshots to fail 409, got %d; body=%s", deleteBase.StatusCode, string(deleteBase.RawBody))
	}

	deleteSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete share snapshot returned error: %v", err)
	}
	if deleteSnapshot.StatusCode != http.StatusAccepted {
		t.Fatalf("expected individual share snapshot delete to succeed, got %d; body=%s", deleteSnapshot.StatusCode, string(deleteSnapshot.RawBody))
	}

	secondSnapshotResp, _ := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if secondSnapshotResp.Headers["x-ms-snapshot"] == "" {
		t.Fatalf("expected second snapshot to be created, got status=%d headers=%v", secondSnapshotResp.StatusCode, secondSnapshotResp.Headers)
	}

	deleteInclude, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version":          "2023-11-03",
		"x-ms-delete-snapshots": "include",
	}))
	if err != nil {
		t.Fatalf("delete share include snapshots returned error: %v", err)
	}
	if deleteInclude.StatusCode != http.StatusAccepted {
		t.Fatalf("expected include-snapshots share delete to succeed, got %d; body=%s", deleteInclude.StatusCode, string(deleteInclude.RawBody))
	}
}

func TestFileDataPlaneSetAndGetShareMetadataReplacesExistingMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	create, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-meta-owner":  "base",
		"x-ms-meta-remove": "yes",
	}))
	if err != nil {
		t.Fatalf("create share returned error: %v", err)
	}
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot share returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "updated",
	}))
	if err != nil {
		t.Fatalf("set share metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusOK || setMetadata.Headers["ETag"] == create.Headers["ETag"] {
		t.Fatalf("expected set share metadata to return 200 with a new ETag, got status=%d headers=%v create=%v", setMetadata.StatusCode, setMetadata.Headers, create.Headers)
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share metadata returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK || len(getMetadata.RawBody) != 0 || getMetadata.Headers["x-ms-meta-owner"] != "updated" || getMetadata.Headers["x-ms-meta-remove"] != "" {
		t.Fatalf("expected replacement metadata headers only, got status=%d body=%q headers=%v", getMetadata.StatusCode, string(getMetadata.RawBody), getMetadata.Headers)
	}

	getSnapshotMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=metadata&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot share metadata returned error: %v", err)
	}
	if getSnapshotMetadata.StatusCode != http.StatusOK || getSnapshotMetadata.Headers["x-ms-snapshot"] != snapshot || getSnapshotMetadata.Headers["x-ms-meta-owner"] != "base" || getSnapshotMetadata.Headers["x-ms-meta-remove"] != "yes" {
		t.Fatalf("expected snapshot metadata to remain point-in-time, got status=%d headers=%v", getSnapshotMetadata.StatusCode, getSnapshotMetadata.Headers)
	}

	setSnapshotMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=metadata&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "blocked",
	}))
	if err != nil {
		t.Fatalf("set snapshot share metadata returned error: %v", err)
	}
	if setSnapshotMetadata.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected setting snapshot metadata to fail 400, got %d; body=%s", setSnapshotMetadata.StatusCode, string(setSnapshotMetadata.RawBody))
	}
}

func TestFileDataPlaneSetSharePropertiesUpdatesShareHeaders(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	create, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-share-quota": "100",
		"x-ms-access-tier": "Hot",
	}))
	if err != nil {
		t.Fatalf("create share returned error: %v", err)
	}
	snapshotResp, _ := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	snapshot := snapshotResp.Headers["x-ms-snapshot"]

	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=properties", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-share-quota": "250",
		"x-ms-access-tier": "Cool",
		"x-ms-root-squash": "RootSquash",
		"x-ms-enable-snapshot-virtual-directory-access": "false",
	}))
	if err != nil {
		t.Fatalf("set share properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusOK || setProperties.Headers["ETag"] == create.Headers["ETag"] {
		t.Fatalf("expected set share properties to return 200 with a new ETag, got status=%d headers=%v create=%v", setProperties.StatusCode, setProperties.Headers, create.Headers)
	}

	getProperties, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share properties returned error: %v", err)
	}
	expected := map[string]string{
		"x-ms-share-quota": "250",
		"x-ms-access-tier": "Cool",
		"x-ms-root-squash": "RootSquash",
		"x-ms-enable-snapshot-virtual-directory-access": "false",
	}
	for key, want := range expected {
		if getProperties.Headers[key] != want {
			t.Fatalf("expected share property %s=%q, got headers=%v", key, want, getProperties.Headers)
		}
	}

	setSnapshotProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=properties&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-share-quota": "300",
	}))
	if err != nil {
		t.Fatalf("set snapshot share properties returned error: %v", err)
	}
	if setSnapshotProperties.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected setting snapshot properties to fail 400, got %d; body=%s", setSnapshotProperties.StatusCode, string(setSnapshotProperties.RawBody))
	}
}

func TestFileDataPlaneLeaseShareAcquireReleaseAndMutationEnforcement(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "base",
	}))
	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head share before lease returned error: %v", err)
	}

	leaseID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("acquire share lease returned error: %v", err)
	}
	if acquire.StatusCode != http.StatusCreated ||
		acquire.Headers["x-ms-lease-id"] != leaseID ||
		acquire.Headers["ETag"] != headBefore.Headers["ETag"] ||
		acquire.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected share lease acquire status 201 with lease ID and unchanged share headers, got status=%d headers=%v before=%v", acquire.StatusCode, acquire.Headers, headBefore.Headers)
	}

	headLeased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head leased share returned error: %v", err)
	}
	if headLeased.Headers["x-ms-lease-status"] != "locked" || headLeased.Headers["x-ms-lease-state"] != "leased" {
		t.Fatalf("expected leased share to report locked/leased, got %v", headLeased.Headers)
	}

	setMetadataWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "blocked",
	}))
	if err != nil {
		t.Fatalf("set share metadata without lease returned error: %v", err)
	}
	if setMetadataWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set share metadata without lease to fail 412, got %d body=%s", setMetadataWithoutLease.StatusCode, string(setMetadataWithoutLease.RawBody))
	}

	setMetadataWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-lease-id":   leaseID,
		"x-ms-meta-owner": "allowed",
	}))
	if err != nil {
		t.Fatalf("set share metadata with lease returned error: %v", err)
	}
	if setMetadataWithLease.StatusCode != http.StatusOK {
		t.Fatalf("expected set share metadata with lease to succeed, got %d body=%s", setMetadataWithLease.StatusCode, string(setMetadataWithLease.RawBody))
	}

	setPropertiesWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=properties", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-share-quota": "200",
	}))
	if err != nil {
		t.Fatalf("set share properties without lease returned error: %v", err)
	}
	if setPropertiesWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set share properties without lease to fail 412, got %d body=%s", setPropertiesWithoutLease.StatusCode, string(setPropertiesWithoutLease.RawBody))
	}

	setPropertiesWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=properties", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-lease-id":    leaseID,
		"x-ms-share-quota": "200",
	}))
	if err != nil {
		t.Fatalf("set share properties with lease returned error: %v", err)
	}
	if setPropertiesWithLease.StatusCode != http.StatusOK || setPropertiesWithLease.Headers["x-ms-share-quota"] != "200" {
		t.Fatalf("expected set share properties with lease to succeed, got status=%d headers=%v body=%s", setPropertiesWithLease.StatusCode, setPropertiesWithLease.Headers, string(setPropertiesWithLease.RawBody))
	}

	deleteWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete share without lease returned error: %v", err)
	}
	if deleteWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected delete share without lease to fail 412, got %d body=%s", deleteWithoutLease.StatusCode, string(deleteWithoutLease.RawBody))
	}

	release, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-lease-action": "release",
		"x-ms-lease-id":     leaseID,
	}))
	if err != nil {
		t.Fatalf("release share lease returned error: %v", err)
	}
	if release.StatusCode != http.StatusOK || release.Headers["ETag"] != setPropertiesWithLease.Headers["ETag"] {
		t.Fatalf("expected share lease release status 200 with current ETag, got status=%d headers=%v setProperties=%v", release.StatusCode, release.Headers, setPropertiesWithLease.Headers)
	}

	deleteAfterRelease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete share after release returned error: %v", err)
	}
	if deleteAfterRelease.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete share after release to succeed, got %d body=%s", deleteAfterRelease.StatusCode, string(deleteAfterRelease.RawBody))
	}
}

func TestFileDataPlaneLeaseShareRenewChangeAndBreak(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	leaseA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	leaseB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	leaseC := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseA,
	}))
	if err != nil {
		t.Fatalf("acquire share lease returned error: %v", err)
	}

	renew, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-lease-action": "renew",
		"x-ms-lease-id":     leaseA,
	}))
	if err != nil {
		t.Fatalf("renew share lease returned error: %v", err)
	}
	if renew.StatusCode != http.StatusOK || renew.Headers["x-ms-lease-id"] != leaseA || renew.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected renew share lease status 200 with active lease and unchanged ETag, got status=%d headers=%v acquire=%v", renew.StatusCode, renew.Headers, acquire.Headers)
	}

	change, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "change",
		"x-ms-lease-id":          leaseA,
		"x-ms-proposed-lease-id": leaseB,
	}))
	if err != nil {
		t.Fatalf("change share lease returned error: %v", err)
	}
	if change.StatusCode != http.StatusOK || change.Headers["x-ms-lease-id"] != leaseB || change.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected change share lease status 200 with proposed lease and unchanged ETag, got status=%d headers=%v acquire=%v", change.StatusCode, change.Headers, acquire.Headers)
	}

	setMetadataWithOldLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-lease-id":   leaseA,
		"x-ms-meta-owner": "old",
	}))
	if err != nil {
		t.Fatalf("set share metadata with old lease returned error: %v", err)
	}
	if setMetadataWithOldLease.StatusCode != http.StatusConflict {
		t.Fatalf("expected old share lease write to fail 409, got %d body=%s", setMetadataWithOldLease.StatusCode, string(setMetadataWithOldLease.RawBody))
	}

	breakLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":            "2023-11-03",
		"x-ms-lease-action":       "break",
		"x-ms-lease-break-period": "0",
	}))
	if err != nil {
		t.Fatalf("break share lease returned error: %v", err)
	}
	if breakLease.StatusCode != http.StatusAccepted || breakLease.Headers["x-ms-lease-time"] != "0" || breakLease.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected break share lease status 202 with lease time 0 and unchanged ETag, got status=%d headers=%v acquire=%v", breakLease.StatusCode, breakLease.Headers, acquire.Headers)
	}

	headBroken, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head broken share lease returned error: %v", err)
	}
	if headBroken.Headers["x-ms-lease-status"] != "unlocked" || headBroken.Headers["x-ms-lease-state"] != "broken" {
		t.Fatalf("expected broken share lease headers, got %v", headBroken.Headers)
	}

	acquireAfterBreak, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseC,
	}))
	if err != nil {
		t.Fatalf("acquire share lease after break returned error: %v", err)
	}
	if acquireAfterBreak.StatusCode != http.StatusCreated || acquireAfterBreak.Headers["x-ms-lease-id"] != leaseC {
		t.Fatalf("expected acquire after break to succeed with new share lease, got status=%d headers=%v", acquireAfterBreak.StatusCode, acquireAfterBreak.Headers)
	}
}

func TestFileDataPlaneLeaseShareSnapshotDeleteEnforcement(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot share returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected share snapshot response to include x-ms-snapshot, got status=%d headers=%v", snapshotResp.StatusCode, snapshotResp.Headers)
	}
	snapshotURL := baseURL + "?restype=share&sharesnapshot=" + url.QueryEscape(snapshot)
	headSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodHead, snapshotURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head share snapshot before lease returned error: %v", err)
	}

	leaseID := "eeeeeeee-eeee-eeee-eeee-eeeeeeeeeeee"
	acquireSnapshotLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, snapshotURL+"&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("acquire share snapshot lease returned error: %v", err)
	}
	if acquireSnapshotLease.StatusCode != http.StatusCreated ||
		acquireSnapshotLease.Headers["x-ms-lease-id"] != leaseID ||
		acquireSnapshotLease.Headers["ETag"] != headSnapshot.Headers["ETag"] ||
		acquireSnapshotLease.Headers["Last-Modified"] != headSnapshot.Headers["Last-Modified"] {
		t.Fatalf("expected share snapshot lease acquire to preserve snapshot headers, got status=%d headers=%v snapshot=%v", acquireSnapshotLease.StatusCode, acquireSnapshotLease.Headers, headSnapshot.Headers)
	}

	headLeasedSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodHead, snapshotURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head leased share snapshot returned error: %v", err)
	}
	if headLeasedSnapshot.Headers["x-ms-snapshot"] != snapshot || headLeasedSnapshot.Headers["x-ms-lease-status"] != "locked" || headLeasedSnapshot.Headers["x-ms-lease-state"] != "leased" {
		t.Fatalf("expected leased share snapshot headers, got %v", headLeasedSnapshot.Headers)
	}

	headBase, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head base share after snapshot lease returned error: %v", err)
	}
	if headBase.Headers["x-ms-lease-status"] != "unlocked" || headBase.Headers["x-ms-lease-state"] != "available" {
		t.Fatalf("expected snapshot lease not to lock base share, got %v", headBase.Headers)
	}

	deleteWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, snapshotURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete leased share snapshot without lease returned error: %v", err)
	}
	if deleteWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected delete leased share snapshot without lease to fail 412, got %d body=%s", deleteWithoutLease.StatusCode, string(deleteWithoutLease.RawBody))
	}

	deleteWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, snapshotURL, nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("delete leased share snapshot with lease returned error: %v", err)
	}
	if deleteWithLease.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete leased share snapshot with lease to succeed, got %d body=%s", deleteWithLease.StatusCode, string(deleteWithLease.RawBody))
	}
}

func TestFileDataPlaneGetShareStatsReturnsUsageBytesAndRejectsSnapshots(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	firstFileURL := baseURL + "/logs/today.txt"
	secondFileURL := baseURL + "/logs/archive.bin"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	headBeforeFiles, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head share before files returned error: %v", err)
	}
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, firstFileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, firstFileURL+"?comp=range", []byte("hello"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, secondFileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "10",
	}))

	stats, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=stats", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share stats returned error: %v", err)
	}
	if stats.StatusCode != http.StatusOK || stats.RawContentType != "application/xml" {
		t.Fatalf("expected share stats status 200 XML, got status=%d contentType=%q body=%s", stats.StatusCode, stats.RawContentType, string(stats.RawBody))
	}
	body := string(stats.RawBody)
	if !strings.Contains(body, "<ShareStats>") || !strings.Contains(body, "<ShareUsageBytes>15</ShareUsageBytes>") {
		t.Fatalf("expected share stats to report 15 usage bytes, got %s", body)
	}
	if stats.Headers["ETag"] != headBeforeFiles.Headers["ETag"] || stats.Headers["Last-Modified"] != headBeforeFiles.Headers["Last-Modified"] {
		t.Fatalf("expected file operations not to mutate share headers reported by stats, stats=%v before=%v", stats.Headers, headBeforeFiles.Headers)
	}

	leaseID := "ffffffff-ffff-ffff-ffff-ffffffffffff"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	statsNoLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=stats", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get leased share stats without lease ID returned error: %v", err)
	}
	if statsNoLease.StatusCode != http.StatusOK {
		t.Fatalf("expected stats without lease ID to succeed on leased share, got %d body=%s", statsNoLease.StatusCode, string(statsNoLease.RawBody))
	}

	statsWrongLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=stats", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": "00000000-0000-0000-0000-000000000000",
	}))
	if err != nil {
		t.Fatalf("get share stats with wrong lease returned error: %v", err)
	}
	if statsWrongLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected stats with wrong lease to fail 412, got %d body=%s", statsWrongLease.StatusCode, string(statsWrongLease.RawBody))
	}

	statsWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=stats", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("get share stats with lease returned error: %v", err)
	}
	if statsWithLease.StatusCode != http.StatusOK || !strings.Contains(string(statsWithLease.RawBody), "<ShareUsageBytes>15</ShareUsageBytes>") {
		t.Fatalf("expected stats with matching lease to succeed, got %d body=%s", statsWithLease.StatusCode, string(statsWithLease.RawBody))
	}

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("snapshot leased share returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	snapshotStats, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=stats&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share snapshot stats returned error: %v", err)
	}
	if snapshotStats.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected share snapshot stats to fail 400, got %d body=%s", snapshotStats.StatusCode, string(snapshotStats.RawBody))
	}
}

func TestFileDataPlaneSetAndGetShareACLReplacesPolicies(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head share before ACL returned error: %v", err)
	}

	firstACL := []byte(`<?xml version="1.0" encoding="utf-8"?>
<SignedIdentifiers>
  <SignedIdentifier>
    <Id>readers</Id>
    <AccessPolicy>
      <Start>2026-06-16T00:00:00.0000000Z</Start>
      <Expiry>2026-06-17T00:00:00.0000000Z</Expiry>
      <Permission>r</Permission>
    </AccessPolicy>
  </SignedIdentifier>
  <SignedIdentifier>
    <Id>writers</Id>
    <AccessPolicy>
      <Start>2026-06-16T01:00:00.0000000Z</Start>
      <Expiry>2026-06-17T01:00:00.0000000Z</Expiry>
      <Permission>rwd</Permission>
    </AccessPolicy>
  </SignedIdentifier>
</SignedIdentifiers>`)
	setACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", firstACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set share ACL returned error: %v", err)
	}
	if setACL.StatusCode != http.StatusOK || setACL.Headers["ETag"] == "" || setACL.Headers["ETag"] == headBefore.Headers["ETag"] || setACL.Headers["Last-Modified"] == "" {
		t.Fatalf("expected set share ACL to update share headers, status=%d headers=%v before=%v", setACL.StatusCode, setACL.Headers, headBefore.Headers)
	}

	getACL, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get share ACL returned error: %v", err)
	}
	if getACL.StatusCode != http.StatusOK || getACL.RawContentType != "application/xml" {
		t.Fatalf("expected get share ACL status 200 XML, got status=%d contentType=%q body=%s", getACL.StatusCode, getACL.RawContentType, string(getACL.RawBody))
	}
	body := string(getACL.RawBody)
	for _, fragment := range []string{
		"<SignedIdentifiers>",
		"<Id>readers</Id>",
		"<Permission>r</Permission>",
		"<Id>writers</Id>",
		"<Permission>rwd</Permission>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected share ACL body to include %q, got %s", fragment, body)
		}
	}
	if getACL.Headers["ETag"] != setACL.Headers["ETag"] || getACL.Headers["Last-Modified"] != setACL.Headers["Last-Modified"] {
		t.Fatalf("expected get ACL to return current share headers, get=%v set=%v", getACL.Headers, setACL.Headers)
	}

	headACL, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?restype=share&comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head share ACL returned error: %v", err)
	}
	if headACL.StatusCode != http.StatusOK || len(headACL.RawBody) != 0 || headACL.Headers["ETag"] != setACL.Headers["ETag"] {
		t.Fatalf("expected HEAD share ACL headers without body, status=%d headers=%v body=%q", headACL.StatusCode, headACL.Headers, string(headACL.RawBody))
	}

	replacementACL := []byte(`<?xml version="1.0" encoding="utf-8"?>
<SignedIdentifiers>
  <SignedIdentifier>
    <Id>auditors</Id>
    <AccessPolicy>
      <Start>2026-06-18T00:00:00.0000000Z</Start>
      <Expiry>2026-06-19T00:00:00.0000000Z</Expiry>
      <Permission>rl</Permission>
    </AccessPolicy>
  </SignedIdentifier>
</SignedIdentifiers>`)
	replaceACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", replacementACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("replace share ACL returned error: %v", err)
	}
	if replaceACL.StatusCode != http.StatusOK || replaceACL.Headers["ETag"] == setACL.Headers["ETag"] {
		t.Fatalf("expected replacing share ACL to rotate ETag, status=%d headers=%v previous=%v", replaceACL.StatusCode, replaceACL.Headers, setACL.Headers)
	}
	replaced, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get replaced share ACL returned error: %v", err)
	}
	replacedBody := string(replaced.RawBody)
	if !strings.Contains(replacedBody, "<Id>auditors</Id>") || strings.Contains(replacedBody, "<Id>readers</Id>") || strings.Contains(replacedBody, "<Id>writers</Id>") {
		t.Fatalf("expected set share ACL replacement semantics, got %s", replacedBody)
	}
}

func TestFileDataPlaneShareACLLeaseLimitAndSnapshotValidation(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	validACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>policy</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)
	setWithAbsentLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", validACL, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": "00000000-0000-0000-0000-000000000000",
	}))
	if err != nil {
		t.Fatalf("set ACL with absent lease returned error: %v", err)
	}
	if setWithAbsentLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set ACL with lease ID on available share to fail 412, got %d body=%s", setWithAbsentLease.StatusCode, string(setWithAbsentLease.RawBody))
	}

	leaseID := "12121212-1212-1212-1212-121212121212"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=lease", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	setWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", validACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set ACL without active lease ID returned error: %v", err)
	}
	if setWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected set ACL without active lease ID to fail 412, got %d body=%s", setWithoutLease.StatusCode, string(setWithoutLease.RawBody))
	}

	setWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", validACL, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("set ACL with active lease returned error: %v", err)
	}
	if setWithLease.StatusCode != http.StatusOK {
		t.Fatalf("expected set ACL with active lease to succeed, got %d body=%s", setWithLease.StatusCode, string(setWithLease.RawBody))
	}

	getWrongLease, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=acl", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": "00000000-0000-0000-0000-000000000000",
	}))
	if err != nil {
		t.Fatalf("get ACL with wrong lease returned error: %v", err)
	}
	if getWrongLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected get ACL with wrong lease to fail 412, got %d body=%s", getWrongLease.StatusCode, string(getWrongLease.RawBody))
	}

	tooManyACL := []byte(`<SignedIdentifiers>
<SignedIdentifier><Id>p1</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p2</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p3</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p4</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p5</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p6</Id><AccessPolicy><Start>2026-06-16T00:00:00.0000000Z</Start><Expiry>2026-06-17T00:00:00.0000000Z</Expiry><Permission>r</Permission></AccessPolicy></SignedIdentifier>
</SignedIdentifiers>`)
	setTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl", tooManyACL, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("set too many ACL policies returned error: %v", err)
	}
	if setTooMany.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected more than five access policies to fail 400, got %d body=%s", setTooMany.StatusCode, string(setTooMany.RawBody))
	}
	getAfterTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=acl", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("get ACL after too many policies returned error: %v", err)
	}
	if !strings.Contains(string(getAfterTooMany.RawBody), "<Id>policy</Id>") || strings.Contains(string(getAfterTooMany.RawBody), "<Id>p6</Id>") {
		t.Fatalf("expected invalid ACL update not to replace existing policies, got %s", string(getAfterTooMany.RawBody))
	}

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version":  "2023-11-03",
		"x-ms-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("snapshot share for ACL validation returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	getSnapshotACL, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=acl&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot ACL returned error: %v", err)
	}
	if getSnapshotACL.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected get snapshot ACL to fail 400, got %d body=%s", getSnapshotACL.StatusCode, string(getSnapshotACL.RawBody))
	}
	setSnapshotACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=acl&sharesnapshot="+url.QueryEscape(snapshot), validACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set snapshot ACL returned error: %v", err)
	}
	if setSnapshotACL.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected set snapshot ACL to fail 400, got %d body=%s", setSnapshotACL.StatusCode, string(setSnapshotACL.RawBody))
	}
}

func TestFileDataPlaneListHandlesReturnsAzureXMLForFilesDirectoriesAndSnapshots(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	directoryURL := baseURL + "/logs"
	fileURL := directoryURL + "/today.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, directoryURL+"?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))

	listFileHandles, err := svc.HandleRequest(storageCtx(t, http.MethodGet, fileURL+"?comp=listhandles", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list file handles returned error: %v", err)
	}
	if listFileHandles.StatusCode != http.StatusOK || listFileHandles.RawContentType != "application/xml" {
		t.Fatalf("expected list file handles status 200 XML, got status=%d contentType=%q body=%s", listFileHandles.StatusCode, listFileHandles.RawContentType, string(listFileHandles.RawBody))
	}
	fileBody := string(listFileHandles.RawBody)
	if !strings.Contains(fileBody, "<EnumerationResults>") || !strings.Contains(fileBody, "<HandleList>") || strings.Contains(fileBody, "<Handle>") {
		t.Fatalf("expected empty handle list XML for file, got %s", fileBody)
	}
	if listFileHandles.Headers["x-ms-version"] == "" {
		t.Fatalf("expected list handles response headers, got %v", listFileHandles.Headers)
	}

	listDirectoryHandles, err := svc.HandleRequest(storageCtx(t, http.MethodGet, directoryURL+"?restype=directory&comp=listhandles&marker=opaque&maxresults=1", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-recursive": "true",
	}))
	if err != nil {
		t.Fatalf("list directory handles returned error: %v", err)
	}
	directoryBody := string(listDirectoryHandles.RawBody)
	for _, fragment := range []string{"<Marker>opaque</Marker>", "<MaxResults>1</MaxResults>", "<HandleList>"} {
		if !strings.Contains(directoryBody, fragment) {
			t.Fatalf("expected directory handle response to include %q, got %s", fragment, directoryBody)
		}
	}

	invalidMaxResults, err := svc.HandleRequest(storageCtx(t, http.MethodGet, directoryURL+"?restype=directory&comp=listhandles&maxresults=0", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list handles with invalid maxresults returned error: %v", err)
	}
	if invalidMaxResults.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid maxresults to fail 400, got %d body=%s", invalidMaxResults.StatusCode, string(invalidMaxResults.RawBody))
	}

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot share for list handles returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	listSnapshotHandles, err := svc.HandleRequest(storageCtx(t, http.MethodGet, fileURL+"?comp=listhandles&sharesnapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list snapshot file handles returned error: %v", err)
	}
	if listSnapshotHandles.StatusCode != http.StatusOK || !strings.Contains(string(listSnapshotHandles.RawBody), "<ShareSnapshot>"+snapshot+"</ShareSnapshot>") {
		t.Fatalf("expected snapshot handle listing to echo share snapshot, got status=%d body=%s", listSnapshotHandles.StatusCode, string(listSnapshotHandles.RawBody))
	}
}

func TestFileDataPlaneForceCloseHandlesValidatesHeadersAndReturnsCounts(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	directoryURL := baseURL + "/logs"
	fileURL := directoryURL + "/today.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, directoryURL+"?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL, nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))

	missingHandleID, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=forceclosehandles", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("force close without handle ID returned error: %v", err)
	}
	if missingHandleID.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing handle ID to fail 400, got %d body=%s", missingHandleID.StatusCode, string(missingHandleID.RawBody))
	}

	recursiveFileClose, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=forceclosehandles", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-handle-id": "*",
		"x-ms-recursive": "true",
	}))
	if err != nil {
		t.Fatalf("force close file recursively returned error: %v", err)
	}
	if recursiveFileClose.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected recursive force close on file to fail 400, got %d body=%s", recursiveFileClose.StatusCode, string(recursiveFileClose.RawBody))
	}

	closeFileHandles, err := svc.HandleRequest(storageCtx(t, http.MethodPut, fileURL+"?comp=forceclosehandles", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-handle-id": "*",
	}))
	if err != nil {
		t.Fatalf("force close file handles returned error: %v", err)
	}
	if closeFileHandles.StatusCode != http.StatusOK || len(closeFileHandles.RawBody) != 0 ||
		closeFileHandles.Headers["x-ms-number-of-handles-closed"] != "0" ||
		closeFileHandles.Headers["x-ms-number-of-handles-failed"] != "0" {
		t.Fatalf("expected force close file handles zero-count response, status=%d headers=%v body=%q", closeFileHandles.StatusCode, closeFileHandles.Headers, string(closeFileHandles.RawBody))
	}

	closeDirectoryHandles, err := svc.HandleRequest(storageCtx(t, http.MethodPut, directoryURL+"?restype=directory&comp=forceclosehandles", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-handle-id": "*",
		"x-ms-recursive": "true",
	}))
	if err != nil {
		t.Fatalf("force close directory handles returned error: %v", err)
	}
	if closeDirectoryHandles.StatusCode != http.StatusOK ||
		closeDirectoryHandles.Headers["x-ms-number-of-handles-closed"] != "0" ||
		closeDirectoryHandles.Headers["x-ms-number-of-handles-failed"] != "0" ||
		closeDirectoryHandles.Headers["x-ms-version"] == "" {
		t.Fatalf("expected force close directory handles zero-count response, got status=%d headers=%v body=%q", closeDirectoryHandles.StatusCode, closeDirectoryHandles.Headers, string(closeDirectoryHandles.RawBody))
	}
}

func TestFileDataPlaneCreateAndGetFilePermissionStoresSDDLAndBinaryFormats(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	sddl := "O:S-1-5-21-1G:S-1-5-21-2D:AI(A;;FA;;;SY)(A;;FA;;;BA)"
	createSDDL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=filepermission", []byte(`{"permission":"`+sddl+`"}`), map[string]string{
		"x-ms-version":           "2023-11-03",
		"Content-Type":           "application/json",
		"x-ms-client-request-id": "permission-sddl",
	}))
	if err != nil {
		t.Fatalf("create SDDL file permission returned error: %v", err)
	}
	sddlKey := createSDDL.Headers["x-ms-file-permission-key"]
	if createSDDL.StatusCode != http.StatusCreated || sddlKey == "" || createSDDL.Headers["x-ms-version"] == "" || len(createSDDL.RawBody) != 0 {
		t.Fatalf("expected create SDDL permission status 201 with permission key and no body, got status=%d headers=%v body=%s", createSDDL.StatusCode, createSDDL.Headers, string(createSDDL.RawBody))
	}

	getSDDL, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=filepermission", nil, map[string]string{
		"x-ms-version":             "2023-11-03",
		"x-ms-file-permission-key": sddlKey,
	}))
	if err != nil {
		t.Fatalf("get SDDL file permission returned error: %v", err)
	}
	sddlBody := decodeJSONBody(t, getSDDL)
	if getSDDL.StatusCode != http.StatusOK || sddlBody["permission"] != sddl {
		t.Fatalf("expected get SDDL permission to return stored descriptor, status=%d body=%v", getSDDL.StatusCode, sddlBody)
	}
	if _, hasFormat := sddlBody["format"]; hasFormat {
		t.Fatalf("expected pre-2024 SDDL permission response without format, got %v", sddlBody)
	}

	binaryPermission := "AQIDBAUG"
	createBinary, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=filepermission", []byte(`{"format":"binary","permission":"`+binaryPermission+`"}`), map[string]string{
		"x-ms-version": "2024-11-04",
		"Content-Type": "application/json",
	}))
	if err != nil {
		t.Fatalf("create binary file permission returned error: %v", err)
	}
	binaryKey := createBinary.Headers["x-ms-file-permission-key"]
	if createBinary.StatusCode != http.StatusCreated || binaryKey == "" || binaryKey == sddlKey {
		t.Fatalf("expected create binary permission to return a distinct key, status=%d headers=%v", createBinary.StatusCode, createBinary.Headers)
	}

	getBinary, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=filepermission", nil, map[string]string{
		"x-ms-version":                "2024-11-04",
		"x-ms-file-permission-key":    binaryKey,
		"x-ms-file-permission-format": "binary",
	}))
	if err != nil {
		t.Fatalf("get binary file permission returned error: %v", err)
	}
	binaryBody := decodeJSONBody(t, getBinary)
	if getBinary.StatusCode != http.StatusOK || binaryBody["format"] != "binary" || binaryBody["permission"] != binaryPermission {
		t.Fatalf("expected get binary permission to return stored binary descriptor, status=%d body=%v", getBinary.StatusCode, binaryBody)
	}
}

func TestFileDataPlaneFilePermissionValidationAndMissingKeys(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	missingPermission, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=filepermission", []byte(`{"format":"sddl"}`), map[string]string{
		"x-ms-version": "2024-11-04",
		"Content-Type": "application/json",
	}))
	if err != nil {
		t.Fatalf("create permission without descriptor returned error: %v", err)
	}
	if missingPermission.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing permission descriptor to fail 400, got %d body=%s", missingPermission.StatusCode, string(missingPermission.RawBody))
	}

	unsupportedFormat, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share&comp=filepermission", []byte(`{"format":"octal","permission":"abc"}`), map[string]string{
		"x-ms-version": "2024-11-04",
		"Content-Type": "application/json",
	}))
	if err != nil {
		t.Fatalf("create permission with unsupported format returned error: %v", err)
	}
	if unsupportedFormat.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unsupported permission format to fail 400, got %d body=%s", unsupportedFormat.StatusCode, string(unsupportedFormat.RawBody))
	}

	missingKey, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=filepermission", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get permission without key returned error: %v", err)
	}
	if missingKey.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected get permission without key to fail 400, got %d body=%s", missingKey.StatusCode, string(missingKey.RawBody))
	}

	unknownKey, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=share&comp=filepermission", nil, map[string]string{
		"x-ms-version":             "2023-11-03",
		"x-ms-file-permission-key": "missing",
	}))
	if err != nil {
		t.Fatalf("get unknown permission key returned error: %v", err)
	}
	if unknownKey.StatusCode != http.StatusNotFound {
		t.Fatalf("expected unknown permission key to fail 404, got %d body=%s", unknownKey.StatusCode, string(unknownKey.RawBody))
	}
}

func TestFileDataPlanePropertiesAndDeletes(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
		"x-ms-content-type":   "text/plain",
		"x-ms-meta-source":    "delete-test",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs/today.txt?comp=range", []byte("hello"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))

	headFile, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head file returned error: %v", err)
	}
	if headFile.StatusCode != http.StatusOK {
		t.Fatalf("expected head file status 200, got %d; body=%s", headFile.StatusCode, string(headFile.RawBody))
	}
	if len(headFile.RawBody) != 0 || headFile.Headers["x-ms-type"] != "File" || headFile.Headers["Content-Length"] != "5" || headFile.Headers["x-ms-meta-source"] != "delete-test" {
		t.Fatalf("expected file properties without body, body=%q headers=%v", string(headFile.RawBody), headFile.Headers)
	}

	deleteNonEmptyDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete non-empty directory returned error: %v", err)
	}
	if deleteNonEmptyDirectory.StatusCode != http.StatusConflict {
		t.Fatalf("expected non-empty directory delete status 409, got %d; body=%s", deleteNonEmptyDirectory.StatusCode, string(deleteNonEmptyDirectory.RawBody))
	}

	deleteFile, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete file returned error: %v", err)
	}
	if deleteFile.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete file status 202, got %d; body=%s", deleteFile.StatusCode, string(deleteFile.RawBody))
	}

	getDeletedFile, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get deleted file returned error: %v", err)
	}
	if getDeletedFile.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted file status 404, got %d; body=%s", getDeletedFile.StatusCode, string(getDeletedFile.RawBody))
	}

	deleteDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete directory returned error: %v", err)
	}
	if deleteDirectory.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete directory status 202, got %d; body=%s", deleteDirectory.StatusCode, string(deleteDirectory.RawBody))
	}
}

func TestFileDataPlaneDirectoryProperties(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-meta-owner":  "platform",
		"x-ms-meta-system": "cloudmock",
	}))

	headDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head directory returned error: %v", err)
	}
	if headDirectory.StatusCode != http.StatusOK {
		t.Fatalf("expected head directory status 200, got %d; body=%s", headDirectory.StatusCode, string(headDirectory.RawBody))
	}
	if len(headDirectory.RawBody) != 0 || headDirectory.Headers["x-ms-file-attributes"] != "Directory" || headDirectory.Headers["x-ms-meta-owner"] != "platform" || headDirectory.Headers["x-ms-meta-system"] != "cloudmock" {
		t.Fatalf("expected directory properties without body, body=%q headers=%v", string(headDirectory.RawBody), headDirectory.Headers)
	}

	getDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get directory returned error: %v", err)
	}
	if getDirectory.StatusCode != http.StatusOK {
		t.Fatalf("expected get directory status 200, got %d; body=%s", getDirectory.StatusCode, string(getDirectory.RawBody))
	}
	if len(getDirectory.RawBody) != 0 || getDirectory.Headers["x-ms-file-attributes"] != "Directory" || getDirectory.Headers["x-ms-meta-owner"] != "platform" {
		t.Fatalf("expected get directory properties without body, body=%q headers=%v", string(getDirectory.RawBody), getDirectory.Headers)
	}
}

func TestFileDataPlaneRootDirectoryList(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "7",
		"x-ms-content-type":   "text/plain",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs/today.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
	}))

	listRoot, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?restype=directory&comp=list", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list root returned error: %v", err)
	}
	if listRoot.StatusCode != http.StatusOK {
		t.Fatalf("expected root list status 200, got %d; body=%s", listRoot.StatusCode, string(listRoot.RawBody))
	}
	body := string(listRoot.RawBody)
	for _, want := range []string{
		`ShareName="reports"`,
		"<Directory>",
		"<Name>logs</Name>",
		"<File>",
		"<Name>readme.txt</Name>",
		"<Content-Length>7</Content-Length>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected root directory listing to contain %s, got: %s", want, body)
		}
	}
	if strings.Contains(body, "today.txt") {
		t.Fatalf("expected root directory listing to exclude nested child, got: %s", body)
	}
}

func TestFileDataPlaneSetFileMetadataReplacesExistingMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "7",
		"x-ms-meta-owner":     "old-owner",
		"x-ms-meta-old":       "removed",
	}))
	initialProperties, _ := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	initialETag := initialProperties.Headers["ETag"]

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt?comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "new-owner",
	}))
	if err != nil {
		t.Fatalf("set file metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected set file metadata status 200, got %d; body=%s", setMetadata.StatusCode, string(setMetadata.RawBody))
	}
	if len(setMetadata.RawBody) != 0 || setMetadata.Headers["ETag"] == "" || setMetadata.Headers["ETag"] == initialETag {
		t.Fatalf("expected set file metadata to return new ETag without body, body=%q headers=%v initialETag=%s", string(setMetadata.RawBody), setMetadata.Headers, initialETag)
	}

	headFile, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head file returned error: %v", err)
	}
	if headFile.Headers["x-ms-meta-owner"] != "new-owner" || headFile.Headers["x-ms-meta-old"] != "" {
		t.Fatalf("expected replacement metadata on file properties, headers=%v", headFile.Headers)
	}
}

func TestFileDataPlaneSetDirectoryMetadataReplacesExistingMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-meta-owner":  "old-owner",
		"x-ms-meta-sticky": "removed",
	}))
	initialProperties, _ := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	initialETag := initialProperties.Headers["ETag"]

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory&comp=metadata", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "new-owner",
	}))
	if err != nil {
		t.Fatalf("set directory metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected set directory metadata status 200, got %d; body=%s", setMetadata.StatusCode, string(setMetadata.RawBody))
	}
	if len(setMetadata.RawBody) != 0 || setMetadata.Headers["ETag"] == "" || setMetadata.Headers["ETag"] == initialETag {
		t.Fatalf("expected set directory metadata to return new ETag without body, body=%q headers=%v initialETag=%s", string(setMetadata.RawBody), setMetadata.Headers, initialETag)
	}

	headDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head directory returned error: %v", err)
	}
	if headDirectory.Headers["x-ms-meta-owner"] != "new-owner" || headDirectory.Headers["x-ms-meta-sticky"] != "" {
		t.Fatalf("expected replacement metadata on directory properties, headers=%v", headDirectory.Headers)
	}
}

func TestFileDataPlaneGetFileMetadataReturnsMetadataHeadersOnly(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "5",
		"x-ms-meta-owner":     "platform",
		"x-ms-meta-purpose":   "metadata-read",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt?comp=range", []byte("hello"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-4",
	}))

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/readme.txt?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get file metadata returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get file metadata status 200, got %d; body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if len(getMetadata.RawBody) != 0 || getMetadata.Headers["x-ms-type"] != "File" || getMetadata.Headers["x-ms-meta-owner"] != "platform" || getMetadata.Headers["x-ms-meta-purpose"] != "metadata-read" {
		t.Fatalf("expected file metadata headers without body, body=%q headers=%v", string(getMetadata.RawBody), getMetadata.Headers)
	}
	if getMetadata.Headers["Content-Length"] != "" || getMetadata.Headers["Content-Type"] != "" {
		t.Fatalf("expected file metadata route to omit content headers, got %v", getMetadata.Headers)
	}
}

func TestFileDataPlaneGetDirectoryMetadataReturnsMetadataHeadersOnly(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-meta-owner":     "platform",
		"x-ms-meta-directory": "logs",
	}))

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/logs?restype=directory&comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get directory metadata returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get directory metadata status 200, got %d; body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if len(getMetadata.RawBody) != 0 || getMetadata.Headers["x-ms-type"] != "Directory" || getMetadata.Headers["x-ms-meta-owner"] != "platform" || getMetadata.Headers["x-ms-meta-directory"] != "logs" {
		t.Fatalf("expected directory metadata headers without body, body=%q headers=%v", string(getMetadata.RawBody), getMetadata.Headers)
	}

	headMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory&comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head directory metadata returned error: %v", err)
	}
	if headMetadata.StatusCode != http.StatusOK || len(headMetadata.RawBody) != 0 || headMetadata.Headers["x-ms-meta-directory"] != "logs" {
		t.Fatalf("expected HEAD directory metadata headers without body, status=%d body=%q headers=%v", headMetadata.StatusCode, string(headMetadata.RawBody), headMetadata.Headers)
	}
}

func TestFileDataPlaneSetDirectoryPropertiesUpdatesSystemHeaders(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	initialProperties, _ := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	initialETag := initialProperties.Headers["ETag"]

	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/logs?restype=directory&comp=properties", nil, map[string]string{
		"x-ms-version":              "2023-11-03",
		"x-ms-file-attributes":      "Directory|Archive",
		"x-ms-file-creation-time":   "2026-06-16T20:00:00Z",
		"x-ms-file-last-write-time": "2026-06-16T20:01:00Z",
		"x-ms-file-change-time":     "2026-06-16T20:02:00Z",
	}))
	if err != nil {
		t.Fatalf("set directory properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusOK {
		t.Fatalf("expected set directory properties status 200, got %d; body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}
	if len(setProperties.RawBody) != 0 || setProperties.Headers["ETag"] == "" || setProperties.Headers["ETag"] == initialETag {
		t.Fatalf("expected set directory properties to return new ETag without body, body=%q headers=%v initialETag=%s", string(setProperties.RawBody), setProperties.Headers, initialETag)
	}
	if setProperties.Headers["x-ms-file-attributes"] != "Directory|Archive" || setProperties.Headers["x-ms-file-change-time"] != "2026-06-16T20:02:00Z" {
		t.Fatalf("expected set directory properties response headers, got %v", setProperties.Headers)
	}

	headDirectory, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/logs?restype=directory", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head directory returned error: %v", err)
	}
	if headDirectory.Headers["x-ms-file-attributes"] != "Directory|Archive" || headDirectory.Headers["x-ms-file-creation-time"] != "2026-06-16T20:00:00Z" || headDirectory.Headers["x-ms-file-last-write-time"] != "2026-06-16T20:01:00Z" || headDirectory.Headers["x-ms-file-change-time"] != "2026-06-16T20:02:00Z" {
		t.Fatalf("expected updated directory system properties on HEAD, got %v", headDirectory.Headers)
	}
}

func TestFileDataPlaneSetFilePropertiesUpdatesHeadersAndResizes(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-file/reports"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=share", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version":        "2023-11-03",
		"x-ms-type":           "file",
		"x-ms-content-length": "11",
		"x-ms-content-type":   "text/plain",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt?comp=range", []byte("hello azure"), map[string]string{
		"x-ms-version": "2023-11-03",
		"x-ms-write":   "update",
		"x-ms-range":   "bytes=0-10",
	}))
	initialProperties, _ := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	initialETag := initialProperties.Headers["ETag"]

	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/readme.txt?comp=properties", nil, map[string]string{
		"x-ms-version":          "2023-11-03",
		"x-ms-content-length":   "5",
		"x-ms-cache-control":    "no-store",
		"x-ms-content-md5":      "CY9rzUYh03PK3k6DJie09g==",
		"x-ms-content-encoding": "gzip",
		"x-ms-content-language": "en-US",
	}))
	if err != nil {
		t.Fatalf("set file properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusOK {
		t.Fatalf("expected set file properties status 200, got %d; body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}
	if len(setProperties.RawBody) != 0 || setProperties.Headers["ETag"] == "" || setProperties.Headers["ETag"] == initialETag {
		t.Fatalf("expected set file properties to return new ETag without body, body=%q headers=%v initialETag=%s", string(setProperties.RawBody), setProperties.Headers, initialETag)
	}

	getFile, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get file returned error: %v", err)
	}
	if getFile.StatusCode != http.StatusOK {
		t.Fatalf("expected get file status 200, got %d; body=%s", getFile.StatusCode, string(getFile.RawBody))
	}
	if string(getFile.RawBody) != "hello" || getFile.Headers["Content-Length"] != "5" {
		t.Fatalf("expected resized file contents and length, body=%q headers=%v", string(getFile.RawBody), getFile.Headers)
	}
	if getFile.Headers["Cache-Control"] != "no-store" || getFile.Headers["Content-MD5"] != "CY9rzUYh03PK3k6DJie09g==" || getFile.Headers["Content-Encoding"] != "gzip" || getFile.Headers["Content-Language"] != "en-US" {
		t.Fatalf("expected updated file HTTP property headers, got %v", getFile.Headers)
	}
	if getFile.Headers["Content-Type"] != "" || getFile.Headers["Content-Disposition"] != "" {
		t.Fatalf("expected omitted HTTP properties to be cleared, got %v", getFile.Headers)
	}
}

func TestTableDataPlaneCreateTableRejectsInvalidNames(t *testing.T) {
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	for _, tableName := range []string{"12Bad", "ab", "Bad_Name", "Tables", strings.Repeat("A", 64)} {
		t.Run(tableName, func(t *testing.T) {
			svc := storage.New()
			createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"`+tableName+`"}`), headers))
			if err != nil {
				t.Fatalf("create table returned error: %v", err)
			}
			if createTable.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected create table status 400, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
			}
			if !strings.Contains(string(createTable.RawBody), "InvalidInput") {
				t.Fatalf("expected InvalidInput response, got %s", string(createTable.RawBody))
			}
		})
	}
}

func TestTableDataPlaneInsertEntityRejectsInvalidKeys(t *testing.T) {
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	cases := []struct {
		name   string
		entity map[string]any
	}{
		{name: "partition slash", entity: map[string]any{"PartitionKey": "p/1", "RowKey": "r1"}},
		{name: "partition hash", entity: map[string]any{"PartitionKey": "p#1", "RowKey": "r1"}},
		{name: "partition control character", entity: map[string]any{"PartitionKey": "p\u0001", "RowKey": "r1"}},
		{name: "row backslash", entity: map[string]any{"PartitionKey": "p1", "RowKey": `r\1`}},
		{name: "row question mark", entity: map[string]any{"PartitionKey": "p1", "RowKey": "r?1"}},
		{name: "row too long", entity: map[string]any{"PartitionKey": "p1", "RowKey": strings.Repeat("r", 1025)}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := storage.New()
			createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
			if err != nil {
				t.Fatalf("create table returned error: %v", err)
			}
			if createTable.StatusCode != http.StatusCreated {
				t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
			}

			body, err := json.Marshal(tc.entity)
			if err != nil {
				t.Fatalf("marshal entity: %v", err)
			}
			insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", body, headers))
			if err != nil {
				t.Fatalf("insert entity returned error: %v", err)
			}
			if insertEntity.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected insert entity status 400, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
			}
			if !strings.Contains(string(insertEntity.RawBody), "InvalidInput") {
				t.Fatalf("expected InvalidInput response, got %s", string(insertEntity.RawBody))
			}

			queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()", nil, headers))
			if err != nil {
				t.Fatalf("query entities returned error: %v", err)
			}
			listed := decodeJSONBody(t, queryEntities)
			if len(listed["value"].([]any)) != 0 {
				t.Fatalf("expected invalid insert to leave table empty, got %v", listed)
			}
		})
	}
}

func TestTableDataPlaneRejectsTooManyEntityProperties(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"
	entityWithCustomProperties := func(count int) map[string]any {
		entity := map[string]any{"PartitionKey": "p1", "RowKey": "r1"}
		for i := 0; i < count; i++ {
			entity[fmt.Sprintf("Prop%03d", i)] = i
		}
		return entity
	}

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	tooManyBody, err := json.Marshal(entityWithCustomProperties(253))
	if err != nil {
		t.Fatalf("marshal oversized entity: %v", err)
	}
	insertTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", tooManyBody, headers))
	if err != nil {
		t.Fatalf("insert oversized entity returned error: %v", err)
	}
	if insertTooMany.StatusCode != http.StatusBadRequest || !strings.Contains(string(insertTooMany.RawBody), "InvalidInput") {
		t.Fatalf("expected oversized insert to return 400 InvalidInput, got %d body=%s", insertTooMany.StatusCode, string(insertTooMany.RawBody))
	}

	queryEmpty, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()", nil, headers))
	if err != nil {
		t.Fatalf("query after rejected oversized insert returned error: %v", err)
	}
	empty := decodeJSONBody(t, queryEmpty)
	if len(empty["value"].([]any)) != 0 {
		t.Fatalf("expected rejected oversized insert to leave table empty, got %v", empty)
	}

	maxBody, err := json.Marshal(entityWithCustomProperties(252))
	if err != nil {
		t.Fatalf("marshal max entity: %v", err)
	}
	insertMax, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", maxBody, headers))
	if err != nil {
		t.Fatalf("insert max entity returned error: %v", err)
	}
	if insertMax.StatusCode != http.StatusCreated {
		t.Fatalf("expected max-property insert status 201, got %d body=%s", insertMax.StatusCode, string(insertMax.RawBody))
	}

	mergeHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	mergeTooMany, err := svc.HandleRequest(storageCtx(t, "MERGE", baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", []byte(`{"Overflow":"nope"}`), mergeHeaders))
	if err != nil {
		t.Fatalf("merge oversized entity returned error: %v", err)
	}
	if mergeTooMany.StatusCode != http.StatusBadRequest || !strings.Contains(string(mergeTooMany.RawBody), "InvalidInput") {
		t.Fatalf("expected oversized merge to return 400 InvalidInput, got %d body=%s", mergeTooMany.StatusCode, string(mergeTooMany.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", nil, headers))
	if err != nil {
		t.Fatalf("get entity after rejected oversized merge returned error: %v", err)
	}
	stored := decodeJSONBody(t, getEntity)
	if _, ok := stored["Overflow"]; ok {
		t.Fatalf("expected rejected oversized merge to preserve entity without Overflow, got %v", stored)
	}
}

func TestTableDataPlaneInsertEntityValidatesPropertyNames(t *testing.T) {
	baseURL := "https://acctest.table.core.windows.net"

	cases := []struct {
		name     string
		version  string
		property string
		want     int
	}{
		{name: "modern version rejects dash", version: "2023-11-03", property: "Bad-Name", want: http.StatusBadRequest},
		{name: "property name too long", version: "2023-11-03", property: strings.Repeat("P", 256), want: http.StatusBadRequest},
		{name: "legacy version allows dash", version: "2009-04-13", property: "Bad-Name", want: http.StatusCreated},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := storage.New()
			headers := map[string]string{
				"x-ms-version": tc.version,
				"Accept":       "application/json;odata=nometadata",
				"Content-Type": "application/json",
			}
			createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
			if err != nil {
				t.Fatalf("create table returned error: %v", err)
			}
			if createTable.StatusCode != http.StatusCreated {
				t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
			}

			entity := map[string]any{
				"PartitionKey": "p1",
				"RowKey":       "r1",
				tc.property:    "value",
			}
			body, err := json.Marshal(entity)
			if err != nil {
				t.Fatalf("marshal entity: %v", err)
			}
			insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", body, headers))
			if err != nil {
				t.Fatalf("insert entity returned error: %v", err)
			}
			if insertEntity.StatusCode != tc.want {
				t.Fatalf("expected insert entity status %d, got %d; body=%s", tc.want, insertEntity.StatusCode, string(insertEntity.RawBody))
			}
			if tc.want == http.StatusBadRequest && !strings.Contains(string(insertEntity.RawBody), "InvalidInput") {
				t.Fatalf("expected InvalidInput response, got %s", string(insertEntity.RawBody))
			}
		})
	}
}

func TestTableDataPlaneRejectsOversizedStringValues(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	oversizedEntity := map[string]any{
		"PartitionKey": "p1",
		"RowKey":       "r1",
		"Description":  strings.Repeat("x", 32769),
	}
	oversizedBody, err := json.Marshal(oversizedEntity)
	if err != nil {
		t.Fatalf("marshal oversized entity: %v", err)
	}
	insertOversized, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", oversizedBody, headers))
	if err != nil {
		t.Fatalf("insert oversized string entity returned error: %v", err)
	}
	if insertOversized.StatusCode != http.StatusBadRequest || !strings.Contains(string(insertOversized.RawBody), "InvalidInput") {
		t.Fatalf("expected oversized string insert to return 400 InvalidInput, got %d body=%s", insertOversized.StatusCode, string(insertOversized.RawBody))
	}

	maxEntity := map[string]any{
		"PartitionKey": "p1",
		"RowKey":       "r1",
		"Description":  strings.Repeat("x", 32768),
	}
	maxBody, err := json.Marshal(maxEntity)
	if err != nil {
		t.Fatalf("marshal max string entity: %v", err)
	}
	insertMax, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", maxBody, headers))
	if err != nil {
		t.Fatalf("insert max string entity returned error: %v", err)
	}
	if insertMax.StatusCode != http.StatusCreated {
		t.Fatalf("expected max string insert status 201, got %d body=%s", insertMax.StatusCode, string(insertMax.RawBody))
	}

	mergeHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	mergeOversized, err := svc.HandleRequest(storageCtx(t, "MERGE", baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", oversizedBody, mergeHeaders))
	if err != nil {
		t.Fatalf("merge oversized string entity returned error: %v", err)
	}
	if mergeOversized.StatusCode != http.StatusBadRequest || !strings.Contains(string(mergeOversized.RawBody), "InvalidInput") {
		t.Fatalf("expected oversized string merge to return 400 InvalidInput, got %d body=%s", mergeOversized.StatusCode, string(mergeOversized.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", nil, headers))
	if err != nil {
		t.Fatalf("get entity after rejected oversized merge returned error: %v", err)
	}
	stored := decodeJSONBody(t, getEntity)
	if stored["Description"] != strings.Repeat("x", 32768) {
		t.Fatalf("expected rejected oversized merge to preserve max Description, got %v", stored["Description"])
	}
}

func TestTableDataPlaneRejectsInvalidBinaryValues(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	cases := []struct {
		name  string
		value string
		want  int
	}{
		{name: "invalid base64", value: "not-base64!", want: http.StatusBadRequest},
		{name: "oversized binary", value: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 65537)), want: http.StatusBadRequest},
		{name: "max binary", value: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 65536)), want: http.StatusCreated},
	}

	for i, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			entity := map[string]any{
				"PartitionKey":       "p1",
				"RowKey":             fmt.Sprintf("r%d", i),
				"Payload@odata.type": "Edm.Binary",
				"Payload":            tc.value,
			}
			body, err := json.Marshal(entity)
			if err != nil {
				t.Fatalf("marshal binary entity: %v", err)
			}
			insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", body, headers))
			if err != nil {
				t.Fatalf("insert binary entity returned error: %v", err)
			}
			if insertEntity.StatusCode != tc.want {
				t.Fatalf("expected binary insert status %d, got %d; body=%s", tc.want, insertEntity.StatusCode, string(insertEntity.RawBody))
			}
			if tc.want == http.StatusBadRequest && !strings.Contains(string(insertEntity.RawBody), "InvalidInput") {
				t.Fatalf("expected InvalidInput response, got %s", string(insertEntity.RawBody))
			}
		})
	}
}

func TestTableDataPlaneRejectsEntitiesOverCombinedPropertySizeLimit(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"
	entityWithPropertyValueSize := func(partitionKey, rowKey string, valueSize int) map[string]any {
		entity := map[string]any{"PartitionKey": partitionKey, "RowKey": rowKey}
		for i := 0; i < 252; i++ {
			entity[fmt.Sprintf("Prop%03d", i)] = strings.Repeat("x", valueSize)
		}
		return entity
	}

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	oversizedBody, err := json.Marshal(entityWithPropertyValueSize("p1", "r1", 2100))
	if err != nil {
		t.Fatalf("marshal oversized entity: %v", err)
	}
	insertOversized, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", oversizedBody, headers))
	if err != nil {
		t.Fatalf("insert oversized combined entity returned error: %v", err)
	}
	if insertOversized.StatusCode != http.StatusBadRequest || !strings.Contains(string(insertOversized.RawBody), "InvalidInput") {
		t.Fatalf("expected combined oversized insert to return 400 InvalidInput, got %d body length=%d", insertOversized.StatusCode, len(insertOversized.RawBody))
	}

	queryEmpty, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()", nil, headers))
	if err != nil {
		t.Fatalf("query after rejected combined oversized insert returned error: %v", err)
	}
	empty := decodeJSONBody(t, queryEmpty)
	if len(empty["value"].([]any)) != 0 {
		t.Fatalf("expected rejected combined oversized insert to leave table empty, got %v", empty)
	}

	validBody, err := json.Marshal(entityWithPropertyValueSize("p1", "r1", 2000))
	if err != nil {
		t.Fatalf("marshal valid entity: %v", err)
	}
	insertValid, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", validBody, headers))
	if err != nil {
		t.Fatalf("insert valid combined entity returned error: %v", err)
	}
	if insertValid.StatusCode != http.StatusCreated {
		t.Fatalf("expected valid combined insert status 201, got %d body=%s", insertValid.StatusCode, string(insertValid.RawBody))
	}

	mergeHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	mergeOversized, err := svc.HandleRequest(storageCtx(t, "MERGE", baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", oversizedBody, mergeHeaders))
	if err != nil {
		t.Fatalf("merge oversized combined entity returned error: %v", err)
	}
	if mergeOversized.StatusCode != http.StatusBadRequest || !strings.Contains(string(mergeOversized.RawBody), "InvalidInput") {
		t.Fatalf("expected combined oversized merge to return 400 InvalidInput, got %d body=%s", mergeOversized.StatusCode, string(mergeOversized.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')?$select=Prop000", nil, headers))
	if err != nil {
		t.Fatalf("get entity after rejected combined merge returned error: %v", err)
	}
	stored := decodeJSONBody(t, getEntity)
	if stored["Prop000"] != strings.Repeat("x", 2000) {
		t.Fatalf("expected rejected combined merge to preserve Prop000 length 2000, got %d", len(fmt.Sprint(stored["Prop000"])))
	}
}

func TestTableDataPlaneRejectsInvalidTypedPropertyValues(t *testing.T) {
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	cases := []struct {
		name       string
		properties map[string]any
	}{
		{
			name: "Int32 above range",
			properties: map[string]any{
				"Score@odata.type": "Edm.Int32",
				"Score":            2147483648,
			},
		},
		{
			name: "Int32 fractional",
			properties: map[string]any{
				"Score@odata.type": "Edm.Int32",
				"Score":            1.5,
			},
		},
		{
			name: "Int64 above range",
			properties: map[string]any{
				"Count@odata.type": "Edm.Int64",
				"Count":            "9223372036854775808",
			},
		},
		{
			name: "invalid GUID",
			properties: map[string]any{
				"ExternalID@odata.type": "Edm.Guid",
				"ExternalID":            "not-a-guid",
			},
		},
		{
			name: "DateTime before supported range",
			properties: map[string]any{
				"CreatedAt@odata.type": "Edm.DateTime",
				"CreatedAt":            "1600-12-31T23:59:59Z",
			},
		},
		{
			name: "Boolean string",
			properties: map[string]any{
				"Active@odata.type": "Edm.Boolean",
				"Active":            "true",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := storage.New()
			createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
			if err != nil {
				t.Fatalf("create table returned error: %v", err)
			}
			if createTable.StatusCode != http.StatusCreated {
				t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
			}

			entity := map[string]any{
				"PartitionKey": "p1",
				"RowKey":       "r1",
			}
			for key, value := range tc.properties {
				entity[key] = value
			}
			body, err := json.Marshal(entity)
			if err != nil {
				t.Fatalf("marshal typed entity: %v", err)
			}
			insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", body, headers))
			if err != nil {
				t.Fatalf("insert typed entity returned error: %v", err)
			}
			if insertEntity.StatusCode != http.StatusBadRequest || !strings.Contains(string(insertEntity.RawBody), "InvalidInput") {
				t.Fatalf("expected typed insert to return 400 InvalidInput, got %d body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
			}

			queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()", nil, headers))
			if err != nil {
				t.Fatalf("query after rejected typed insert returned error: %v", err)
			}
			listed := decodeJSONBody(t, queryEntities)
			if len(listed["value"].([]any)) != 0 {
				t.Fatalf("expected rejected typed insert to leave table empty, got %v", listed)
			}
		})
	}
}

func TestTableDataPlaneAcceptsValidTypedPropertyValues(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	entity := map[string]any{
		"PartitionKey":          "p1",
		"RowKey":                "r1",
		"Score@odata.type":      "Edm.Int32",
		"Score":                 2147483647,
		"Count@odata.type":      "Edm.Int64",
		"Count":                 "9223372036854775807",
		"ExternalID@odata.type": "Edm.Guid",
		"ExternalID":            "a455c695-df98-5678-aaaa-81d3367e5a34",
		"CreatedAt@odata.type":  "Edm.DateTime",
		"CreatedAt":             "1601-01-01T00:00:00Z",
		"Active@odata.type":     "Edm.Boolean",
		"Active":                true,
		"Ratio@odata.type":      "Edm.Double",
		"Ratio":                 1.5,
	}
	body, err := json.Marshal(entity)
	if err != nil {
		t.Fatalf("marshal typed entity: %v", err)
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", body, headers))
	if err != nil {
		t.Fatalf("insert typed entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected valid typed insert status 201, got %d body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
}

func TestTableDataPlaneQueryEntitiesSupportsCompoundAndFilters(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","Value":"one","Score":10}`,
		`{"PartitionKey":"p1","RowKey":"r2","Value":"two","Score":20}`,
		`{"PartitionKey":"p1","RowKey":"r3","Value":"three","Score":30}`,
		`{"PartitionKey":"p1","RowKey":"r4","Value":"four","Score":40}`,
		`{"PartitionKey":"p2","RowKey":"r2","Value":"other","Score":20}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	queryURL := baseURL + "/Tasks()?$filter=PartitionKey%20eq%20'p1'%20and%20RowKey%20ge%20'r2'%20and%20RowKey%20lt%20'r4'&$select=Value,Score"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("compound filter query returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two compound-filtered entities, got %v", query)
	}
	expected := []struct {
		value string
		score float64
	}{
		{value: "two", score: 20},
		{value: "three", score: 30},
	}
	for i, want := range expected {
		entity := values[i].(map[string]any)
		if entity["PartitionKey"] != "p1" || entity["Value"] != want.value || entity["Score"] != want.score {
			t.Fatalf("unexpected compound-filtered entity %d: %v", i, entity)
		}
	}
}

func TestTableDataPlaneQueryEntitiesSupportsOrFilters(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","Value":"one","Score":10}`,
		`{"PartitionKey":"p1","RowKey":"r2","Value":"two","Score":20}`,
		`{"PartitionKey":"p2","RowKey":"r1","Value":"three","Score":40}`,
		`{"PartitionKey":"p2","RowKey":"r2","Value":"four","Score":5}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	queryURL := baseURL + "/Tasks()?$filter=PartitionKey%20eq%20'p1'%20or%20Score%20gt%2035&$select=Value,Score"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("or filter query returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 3 {
		t.Fatalf("expected three or-filtered entities, got %v", query)
	}
	expected := []struct {
		value string
		score float64
	}{
		{value: "one", score: 10},
		{value: "two", score: 20},
		{value: "three", score: 40},
	}
	for i, want := range expected {
		entity := values[i].(map[string]any)
		if entity["Value"] != want.value || entity["Score"] != want.score {
			t.Fatalf("unexpected or-filtered entity %d: %v", i, entity)
		}
	}
}

func TestTableDataPlaneQueryEntitiesSupportsBooleanNotFilters(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","Value":"active","IsActive":true}`,
		`{"PartitionKey":"p1","RowKey":"r2","Value":"inactive","IsActive":false}`,
		`{"PartitionKey":"p1","RowKey":"r3","Value":"active-again","IsActive":true}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	queryURL := baseURL + "/Tasks()?$filter=not%20IsActive&$select=Value,IsActive"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("not filter query returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one not-filtered entity, got %v", query)
	}
	entity := values[0].(map[string]any)
	if entity["Value"] != "inactive" || entity["IsActive"] != false {
		t.Fatalf("unexpected not-filtered entity: %v", entity)
	}
}

func TestTableDataPlaneQueryEntitiesSupportsTypedLiteralFilters(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Customers"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","Name":"match","CustomerSince@odata.type":"Edm.DateTime","CustomerSince":"2008-07-10T00:00:00Z","GuidValue@odata.type":"Edm.Guid","GuidValue":"a455c695-df98-5678-aaaa-81d3367e5a34"}`,
		`{"PartitionKey":"p1","RowKey":"r2","Name":"other-date","CustomerSince@odata.type":"Edm.DateTime","CustomerSince":"2009-07-10T00:00:00Z","GuidValue@odata.type":"Edm.Guid","GuidValue":"a455c695-df98-5678-aaaa-81d3367e5a34"}`,
		`{"PartitionKey":"p1","RowKey":"r3","Name":"other-guid","CustomerSince@odata.type":"Edm.DateTime","CustomerSince":"2008-07-10T00:00:00Z","GuidValue@odata.type":"Edm.Guid","GuidValue":"00000000-0000-0000-0000-000000000000"}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Customers", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	queryURL := baseURL + "/Customers()?$filter=CustomerSince%20eq%20datetime'2008-07-10T00%3A00%3A00Z'%20and%20GuidValue%20eq%20guid'a455c695-df98-5678-aaaa-81d3367e5a34'&$select=Name,CustomerSince,GuidValue"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("typed literal filter query returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one typed-literal filtered entity, got %v", query)
	}
	entity := values[0].(map[string]any)
	if entity["Name"] != "match" || entity["CustomerSince"] != "2008-07-10T00:00:00Z" || entity["GuidValue"] != "a455c695-df98-5678-aaaa-81d3367e5a34" {
		t.Fatalf("unexpected typed-literal filtered entity: %v", entity)
	}
}

func TestTableDataPlaneQueryEntitiesSupportsEscapedSingleQuoteFilters(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Customers"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","LastName":"o'clock","Segment":"target"}`,
		`{"PartitionKey":"p1","RowKey":"r2","LastName":"oclock","Segment":"plain"}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Customers", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	queryURL := baseURL + "/Customers()?$filter=LastName%20eq%20'o''clock'&$select=LastName,Segment"
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("escaped quote filter query returned error: %v", err)
	}
	query := decodeJSONBody(t, queryEntities)
	values := query["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one escaped-quote filtered entity, got %v", query)
	}
	entity := values[0].(map[string]any)
	if entity["LastName"] != "o'clock" || entity["Segment"] != "target" {
		t.Fatalf("unexpected escaped-quote filtered entity: %v", entity)
	}
}

func TestTableDataPlaneQueryEntitiesRejectsSelectBeforeSupportedVersion(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	queryHeaders := map[string]string{
		"x-ms-version": "2010-08-18",
		"Accept":       "application/json;odata=nometadata",
	}
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()?$select=Value", nil, queryHeaders))
	if err != nil {
		t.Fatalf("legacy select query returned error: %v", err)
	}
	if queryEntities.StatusCode != http.StatusBadRequest || !strings.Contains(string(queryEntities.RawBody), "InvalidInput") {
		t.Fatalf("expected legacy $select query to return 400 InvalidInput, got %d body=%s", queryEntities.StatusCode, string(queryEntities.RawBody))
	}
}

func TestTableDataPlaneRejectsSelectOverReturnedPropertyLimit(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	fields := make([]string, 0, 256)
	for i := 0; i < 256; i++ {
		fields = append(fields, fmt.Sprintf("Field%03d", i))
	}
	selectQuery := "$select=" + url.QueryEscape(strings.Join(fields, ","))

	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "entity set", url: baseURL + "/Tasks()?" + selectQuery},
		{name: "keyed entity", url: baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')?" + selectQuery},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tc.url, nil, headers))
			if err != nil {
				t.Fatalf("over-limit select request returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(resp.RawBody), "InvalidInput") {
				t.Fatalf("expected over-limit $select to return 400 InvalidInput, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
		})
	}
}

func TestTableDataPlaneGetEntitySupportsSelectProjection(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one","Extra":"hidden"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')?$select=Value", nil, headers))
	if err != nil {
		t.Fatalf("get projected entity returned error: %v", err)
	}
	if getEntity.StatusCode != http.StatusOK {
		t.Fatalf("expected get projected entity status 200, got %d; body=%s", getEntity.StatusCode, string(getEntity.RawBody))
	}
	entity := decodeJSONBody(t, getEntity)
	if entity["Value"] != "one" || entity["PartitionKey"] != "p1" || entity["RowKey"] != "r1" {
		t.Fatalf("unexpected projected keyed entity: %v", entity)
	}
	if _, ok := entity["Extra"]; ok {
		t.Fatalf("expected keyed entity projection to omit unselected Extra property, got %v", entity)
	}
}

func TestTableDataPlaneGetEntitySupportsEscapedQuoteKeys(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p'1","RowKey":"r'1","Value":"quoted"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p''1',RowKey='r''1')", nil, headers))
	if err != nil {
		t.Fatalf("get escaped-key entity returned error: %v", err)
	}
	if getEntity.StatusCode != http.StatusOK {
		t.Fatalf("expected get escaped-key entity status 200, got %d; body=%s", getEntity.StatusCode, string(getEntity.RawBody))
	}
	entity := decodeJSONBody(t, getEntity)
	if entity["PartitionKey"] != "p'1" || entity["RowKey"] != "r'1" || entity["Value"] != "quoted" {
		t.Fatalf("unexpected escaped-key entity: %v", entity)
	}
}

func TestTableDataPlaneQueryEntitiesSelectIncludesMissingPropertiesAsNull(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()?$select=Value,Missing", nil, headers))
	if err != nil {
		t.Fatalf("query projected entity returned error: %v", err)
	}
	if queryEntities.StatusCode != http.StatusOK {
		t.Fatalf("expected query projected entity status 200, got %d; body=%s", queryEntities.StatusCode, string(queryEntities.RawBody))
	}
	body := decodeJSONBody(t, queryEntities)
	values := body["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one projected entity, got %v", body)
	}
	entity := values[0].(map[string]any)
	if entity["Value"] != "one" {
		t.Fatalf("expected selected Value property, got %v", entity)
	}
	if _, ok := entity["Missing"]; !ok {
		t.Fatalf("expected missing selected property to be included with null value, got %v", entity)
	}
	if entity["Missing"] != nil {
		t.Fatalf("expected missing selected property to be null, got %v", entity)
	}
}

func TestTableDataPlaneGetEntitySelectIncludesMissingPropertiesAsNull(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')?$select=Value,Missing", nil, headers))
	if err != nil {
		t.Fatalf("get projected entity returned error: %v", err)
	}
	if getEntity.StatusCode != http.StatusOK {
		t.Fatalf("expected get projected entity status 200, got %d; body=%s", getEntity.StatusCode, string(getEntity.RawBody))
	}
	entity := decodeJSONBody(t, getEntity)
	if entity["Value"] != "one" {
		t.Fatalf("expected selected Value property, got %v", entity)
	}
	if _, ok := entity["Missing"]; !ok {
		t.Fatalf("expected missing selected property to be included with null value, got %v", entity)
	}
	if entity["Missing"] != nil {
		t.Fatalf("expected missing selected property to be null, got %v", entity)
	}
}

func TestTableDataPlaneGetEntityRejectsSelectBeforeSupportedVersion(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	queryHeaders := map[string]string{
		"x-ms-version": "2010-08-18",
		"Accept":       "application/json;odata=nometadata",
	}
	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')?$select=Value", nil, queryHeaders))
	if err != nil {
		t.Fatalf("legacy projected entity returned error: %v", err)
	}
	if getEntity.StatusCode != http.StatusBadRequest || !strings.Contains(string(getEntity.RawBody), "InvalidInput") {
		t.Fatalf("expected legacy keyed $select to return 400 InvalidInput, got %d body=%s", getEntity.StatusCode, string(getEntity.RawBody))
	}
}

func TestTableDataPlaneQueryEntitiesRejectsMoreThanFifteenFilterComparisons(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	clauses := make([]string, 0, 16)
	for i := 0; i < 16; i++ {
		clauses = append(clauses, fmt.Sprintf("Score eq %d", i))
	}
	queryURL := baseURL + "/Tasks()?$filter=" + url.QueryEscape(strings.Join(clauses, " or "))
	queryEntities, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("over-limit filter query returned error: %v", err)
	}
	if queryEntities.StatusCode != http.StatusBadRequest || !strings.Contains(string(queryEntities.RawBody), "InvalidInput") {
		t.Fatalf("expected over-limit entity filter to return 400 InvalidInput, got %d body=%s", queryEntities.StatusCode, string(queryEntities.RawBody))
	}
}

func TestTableDataPlaneRejectsNullFilterLiterals(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "query entities", url: baseURL + "/Tasks()?$filter=Value%20eq%20null"},
		{name: "query tables", url: baseURL + "/Tables?$filter=TableName%20eq%20null"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tc.url, nil, headers))
			if err != nil {
				t.Fatalf("null filter request returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(resp.RawBody), "InvalidInput") {
				t.Fatalf("expected null filter to return 400 InvalidInput, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
		})
	}
}

func TestTableDataPlaneRejectsDynamicRightSideFilterProperties(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Score":3,"OtherScore":3}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "query entities", url: baseURL + "/Tasks()?$filter=Score%20eq%20OtherScore"},
		{name: "query tables", url: baseURL + "/Tables?$filter=TableName%20eq%20OtherName"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tc.url, nil, headers))
			if err != nil {
				t.Fatalf("dynamic filter request returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(resp.RawBody), "InvalidInput") {
				t.Fatalf("expected dynamic right-side filter to return 400 InvalidInput, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
		})
	}
}

func TestTableDataPlaneRejectsUnsupportedODataQueryOptions(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	for _, tc := range []struct {
		name string
		url  string
	}{
		{name: "query entities", url: baseURL + "/Tasks()?$orderby=Value"},
		{name: "query tables", url: baseURL + "/Tables?$orderby=TableName"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tc.url, nil, headers))
			if err != nil {
				t.Fatalf("unsupported OData option request returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(resp.RawBody), "InvalidInput") {
				t.Fatalf("expected unsupported OData option to return 400 InvalidInput, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
		})
	}
}

func TestTableDataPlaneQueryTablesRejectsMoreThanFifteenFilterComparisons(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	clauses := make([]string, 0, 16)
	for i := 0; i < 16; i++ {
		clauses = append(clauses, fmt.Sprintf("TableName ne 'T%02d'", i))
	}
	queryURL := baseURL + "/Tables?$filter=" + url.QueryEscape(strings.Join(clauses, " and "))
	queryTables, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queryURL, nil, headers))
	if err != nil {
		t.Fatalf("over-limit table filter query returned error: %v", err)
	}
	if queryTables.StatusCode != http.StatusBadRequest || !strings.Contains(string(queryTables.RawBody), "InvalidInput") {
		t.Fatalf("expected over-limit table filter to return 400 InvalidInput, got %d body=%s", queryTables.StatusCode, string(queryTables.RawBody))
	}
}

func TestTableDataPlaneQueryTablesPagesWithContinuationToken(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	for _, tableName := range []string{"Gamma", "Alpha", "Omega", "Beta", "Delta"} {
		createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"`+tableName+`"}`), headers))
		if err != nil {
			t.Fatalf("create table %s returned error: %v", tableName, err)
		}
		if createTable.StatusCode != http.StatusCreated {
			t.Fatalf("expected create table %s status 201, got %d; body=%s", tableName, createTable.StatusCode, string(createTable.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tables?$top=2", nil, headers))
	if err != nil {
		t.Fatalf("first table page returned error: %v", err)
	}
	assertTableNamesPage(t, firstPage, []string{"Alpha", "Beta"})
	if firstPage.Headers["x-ms-continuation-NextTableName"] != "Delta" {
		t.Fatalf("expected first table continuation Delta, got headers=%v", firstPage.Headers)
	}

	secondURL := baseURL + "/Tables?$top=2&NextTableName=" + url.QueryEscape(firstPage.Headers["x-ms-continuation-NextTableName"])
	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, secondURL, nil, headers))
	if err != nil {
		t.Fatalf("second table page returned error: %v", err)
	}
	assertTableNamesPage(t, secondPage, []string{"Delta", "Gamma"})
	if secondPage.Headers["x-ms-continuation-NextTableName"] != "Omega" {
		t.Fatalf("expected second table continuation Omega, got headers=%v", secondPage.Headers)
	}

	thirdURL := baseURL + "/Tables?$top=2&NextTableName=" + url.QueryEscape(secondPage.Headers["x-ms-continuation-NextTableName"])
	thirdPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, thirdURL, nil, headers))
	if err != nil {
		t.Fatalf("third table page returned error: %v", err)
	}
	assertTableNamesPage(t, thirdPage, []string{"Omega"})
	if thirdPage.Headers["x-ms-continuation-NextTableName"] != "" {
		t.Fatalf("expected final table page to omit continuation, got headers=%v", thirdPage.Headers)
	}
}

func TestTableDataPlaneQueryTablesDefaultsToThousandTablePages(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	for i := 0; i < 1001; i++ {
		tableName := fmt.Sprintf("Table%04d", i)
		createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"`+tableName+`"}`), headers))
		if err != nil {
			t.Fatalf("create table %s returned error: %v", tableName, err)
		}
		if createTable.StatusCode != http.StatusCreated {
			t.Fatalf("expected create table %s status 201, got %d; body=%s", tableName, createTable.StatusCode, string(createTable.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tables", nil, headers))
	if err != nil {
		t.Fatalf("first default table page returned error: %v", err)
	}
	assertTablePageLength(t, firstPage, 1000)
	if firstPage.Headers["x-ms-continuation-NextTableName"] != "Table1000" {
		t.Fatalf("expected default table continuation Table1000, got headers=%v", firstPage.Headers)
	}

	secondURL := baseURL + "/Tables?NextTableName=" + url.QueryEscape(firstPage.Headers["x-ms-continuation-NextTableName"])
	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, secondURL, nil, headers))
	if err != nil {
		t.Fatalf("second default table page returned error: %v", err)
	}
	assertTablePageLength(t, secondPage, 1)
	if secondPage.Headers["x-ms-continuation-NextTableName"] != "" {
		t.Fatalf("expected final default table page to omit continuation, got headers=%v", secondPage.Headers)
	}
}

func TestTableDataPlaneQueryTablesCapsTopAtThousandTables(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	for i := 0; i < 1001; i++ {
		tableName := fmt.Sprintf("Table%04d", i)
		createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"`+tableName+`"}`), headers))
		if err != nil {
			t.Fatalf("create table %s returned error: %v", tableName, err)
		}
		if createTable.StatusCode != http.StatusCreated {
			t.Fatalf("expected create table %s status 201, got %d; body=%s", tableName, createTable.StatusCode, string(createTable.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tables?$top=1500", nil, headers))
	if err != nil {
		t.Fatalf("top-over-limit table page returned error: %v", err)
	}
	assertTablePageLength(t, firstPage, 1000)
	if firstPage.Headers["x-ms-continuation-NextTableName"] != "Table1000" {
		t.Fatalf("expected top-over-limit table continuation Table1000, got headers=%v", firstPage.Headers)
	}
}

func TestTableDataPlaneQueryTablesFiltersBeforePaging(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	for _, tableName := range []string{"Gamma", "Alpha", "Omega", "Beta", "Delta"} {
		createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"`+tableName+`"}`), headers))
		if err != nil {
			t.Fatalf("create table %s returned error: %v", tableName, err)
		}
		if createTable.StatusCode != http.StatusCreated {
			t.Fatalf("expected create table %s status 201, got %d; body=%s", tableName, createTable.StatusCode, string(createTable.RawBody))
		}
	}

	filter := "$filter=TableName%20ge%20'Delta'"
	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tables?"+filter+"&$top=2", nil, headers))
	if err != nil {
		t.Fatalf("first filtered table page returned error: %v", err)
	}
	assertTableNamesPage(t, firstPage, []string{"Delta", "Gamma"})
	if firstPage.Headers["x-ms-continuation-NextTableName"] != "Omega" {
		t.Fatalf("expected filtered table continuation Omega, got headers=%v", firstPage.Headers)
	}

	secondURL := baseURL + "/Tables?" + filter + "&$top=2&NextTableName=" + url.QueryEscape(firstPage.Headers["x-ms-continuation-NextTableName"])
	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, secondURL, nil, headers))
	if err != nil {
		t.Fatalf("second filtered table page returned error: %v", err)
	}
	assertTableNamesPage(t, secondPage, []string{"Omega"})
	if secondPage.Headers["x-ms-continuation-NextTableName"] != "" {
		t.Fatalf("expected final filtered table page to omit continuation, got headers=%v", secondPage.Headers)
	}
}

func assertTablePageLength(t *testing.T, resp *service.Response, length int) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query tables status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeJSONBody(t, resp)
	rawValues := body["value"].([]any)
	if len(rawValues) != length {
		t.Fatalf("expected %d tables, got %d", length, len(rawValues))
	}
}

func assertTableNamesPage(t *testing.T, resp *service.Response, tableNames []string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query tables status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeJSONBody(t, resp)
	rawValues := body["value"].([]any)
	if len(rawValues) != len(tableNames) {
		t.Fatalf("expected %d tables, got %d body=%v", len(tableNames), len(rawValues), body)
	}
	for i, expected := range tableNames {
		table := rawValues[i].(map[string]any)
		if table["TableName"] != expected {
			t.Fatalf("expected table %d name %q, got %v", i, expected, table)
		}
	}
}

func TestTableDataPlaneQueryEntitiesPagesWithContinuationTokens(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for _, entity := range []string{
		`{"PartitionKey":"p1","RowKey":"r1","Value":"one"}`,
		`{"PartitionKey":"p1","RowKey":"r2","Value":"two"}`,
		`{"PartitionKey":"p1","RowKey":"r3","Value":"three"}`,
		`{"PartitionKey":"p2","RowKey":"r1","Value":"four"}`,
		`{"PartitionKey":"p2","RowKey":"r2","Value":"five"}`,
	} {
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity returned error: %v", err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()?$top=2&$select=Value", nil, headers))
	if err != nil {
		t.Fatalf("first page query returned error: %v", err)
	}
	assertTableEntityPage(t, firstPage, []string{"one", "two"})
	if firstPage.Headers["x-ms-continuation-NextPartitionKey"] != "p1" || firstPage.Headers["x-ms-continuation-NextRowKey"] != "r3" {
		t.Fatalf("expected first continuation to point at p1/r3, got headers=%v", firstPage.Headers)
	}

	secondURL := fmt.Sprintf("%s/Tasks()?$top=2&$select=Value&NextPartitionKey=%s&NextRowKey=%s",
		baseURL,
		url.QueryEscape(firstPage.Headers["x-ms-continuation-NextPartitionKey"]),
		url.QueryEscape(firstPage.Headers["x-ms-continuation-NextRowKey"]),
	)
	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, secondURL, nil, headers))
	if err != nil {
		t.Fatalf("second page query returned error: %v", err)
	}
	assertTableEntityPage(t, secondPage, []string{"three", "four"})
	if secondPage.Headers["x-ms-continuation-NextPartitionKey"] != "p2" || secondPage.Headers["x-ms-continuation-NextRowKey"] != "r2" {
		t.Fatalf("expected second continuation to point at p2/r2, got headers=%v", secondPage.Headers)
	}

	thirdURL := fmt.Sprintf("%s/Tasks()?$top=2&$select=Value&NextPartitionKey=%s&NextRowKey=%s",
		baseURL,
		url.QueryEscape(secondPage.Headers["x-ms-continuation-NextPartitionKey"]),
		url.QueryEscape(secondPage.Headers["x-ms-continuation-NextRowKey"]),
	)
	thirdPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, thirdURL, nil, headers))
	if err != nil {
		t.Fatalf("third page query returned error: %v", err)
	}
	assertTableEntityPage(t, thirdPage, []string{"five"})
	if thirdPage.Headers["x-ms-continuation-NextPartitionKey"] != "" || thirdPage.Headers["x-ms-continuation-NextRowKey"] != "" {
		t.Fatalf("expected final page to omit continuation headers, got headers=%v", thirdPage.Headers)
	}
}

func TestTableDataPlaneQueryEntitiesDefaultsToThousandEntityPages(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for i := 0; i < 1001; i++ {
		entity := fmt.Sprintf(`{"PartitionKey":"p1","RowKey":"r%04d","Value":%d}`, i, i)
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity %d returned error: %v", i, err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity %d status 201, got %d; body=%s", i, insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()?$select=Value", nil, headers))
	if err != nil {
		t.Fatalf("first default entity page returned error: %v", err)
	}
	assertTableEntityPageLength(t, firstPage, 1000)
	if firstPage.Headers["x-ms-continuation-NextPartitionKey"] != "p1" || firstPage.Headers["x-ms-continuation-NextRowKey"] != "r1000" {
		t.Fatalf("expected default-page continuation to point at p1/r1000, got headers=%v", firstPage.Headers)
	}

	secondURL := fmt.Sprintf("%s/Tasks()?$select=Value&NextPartitionKey=%s&NextRowKey=%s",
		baseURL,
		url.QueryEscape(firstPage.Headers["x-ms-continuation-NextPartitionKey"]),
		url.QueryEscape(firstPage.Headers["x-ms-continuation-NextRowKey"]),
	)
	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, secondURL, nil, headers))
	if err != nil {
		t.Fatalf("second default entity page returned error: %v", err)
	}
	assertTableEntityPageLength(t, secondPage, 1)
	if secondPage.Headers["x-ms-continuation-NextPartitionKey"] != "" || secondPage.Headers["x-ms-continuation-NextRowKey"] != "" {
		t.Fatalf("expected final default page to omit continuation headers, got headers=%v", secondPage.Headers)
	}
}

func TestTableDataPlaneQueryEntitiesCapsTopAtThousandEntities(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "https://acctest.table.core.windows.net"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	for i := 0; i < 1001; i++ {
		entity := fmt.Sprintf(`{"PartitionKey":"p1","RowKey":"r%04d","Value":%d}`, i, i)
		insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(entity), headers))
		if err != nil {
			t.Fatalf("insert entity %d returned error: %v", i, err)
		}
		if insertEntity.StatusCode != http.StatusCreated {
			t.Fatalf("expected insert entity %d status 201, got %d; body=%s", i, insertEntity.StatusCode, string(insertEntity.RawBody))
		}
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks()?$top=1500&$select=Value", nil, headers))
	if err != nil {
		t.Fatalf("top-over-limit entity page returned error: %v", err)
	}
	assertTableEntityPageLength(t, firstPage, 1000)
	if firstPage.Headers["x-ms-continuation-NextPartitionKey"] != "p1" || firstPage.Headers["x-ms-continuation-NextRowKey"] != "r1000" {
		t.Fatalf("expected top-over-limit entity continuation to point at p1/r1000, got headers=%v", firstPage.Headers)
	}
}

func assertTableEntityPage(t *testing.T, resp *service.Response, values []string) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query entities status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeJSONBody(t, resp)
	rawValues := body["value"].([]any)
	if len(rawValues) != len(values) {
		t.Fatalf("expected %d entities, got %d body=%v", len(values), len(rawValues), body)
	}
	for i, expected := range values {
		entity := rawValues[i].(map[string]any)
		if entity["Value"] != expected {
			t.Fatalf("expected entity %d value %q, got %v", i, expected, entity)
		}
	}
}

func assertTableEntityPageLength(t *testing.T, resp *service.Response, length int) {
	t.Helper()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected query entities status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeJSONBody(t, resp)
	rawValues := body["value"].([]any)
	if len(rawValues) != length {
		t.Fatalf("expected %d entities, got %d", length, len(rawValues))
	}
}

func TestTableDataPlaneMergeVerbPartiallyUpdatesEntity(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"hello","Score":50}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	oldETag := insertEntity.Headers["ETag"]

	entityURL := baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')"
	mergeHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	mergeEntity, err := svc.HandleRequest(storageCtx(t, "MERGE", entityURL, []byte(`{"Score":80}`), mergeHeaders))
	if err != nil {
		t.Fatalf("merge entity returned error: %v", err)
	}
	if mergeEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected merge entity status 204, got %d; body=%s", mergeEntity.StatusCode, string(mergeEntity.RawBody))
	}
	if mergeEntity.Headers["ETag"] == "" || mergeEntity.Headers["ETag"] == oldETag {
		t.Fatalf("expected merge to return a changed ETag, old=%q new=%q", oldETag, mergeEntity.Headers["ETag"])
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
	if err != nil {
		t.Fatalf("get merged entity returned error: %v", err)
	}
	merged := decodeJSONBody(t, getEntity)
	if merged["Value"] != "hello" || merged["Score"] != float64(80) {
		t.Fatalf("expected MERGE to preserve existing fields and update supplied fields, got %v", merged)
	}
}

func TestTableDataPlaneLegacyUpdateAndMergeRequireIfMatch(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"original","Score":10}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	entityURL := baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')"
	legacyHeaders := map[string]string{
		"x-ms-version": "2009-09-19",
		"Content-Type": "application/json",
	}
	for _, tt := range []struct {
		method string
		body   string
	}{
		{method: http.MethodPut, body: `{"PartitionKey":"p1","RowKey":"r1","Value":"legacy-put","Score":20}`},
		{method: "MERGE", body: `{"Score":30}`},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, tt.method, entityURL, []byte(tt.body), legacyHeaders))
		if err != nil {
			t.Fatalf("%s without If-Match returned error: %v", tt.method, err)
		}
		if resp.StatusCode != http.StatusBadRequest || !strings.Contains(string(resp.RawBody), "InvalidInput") {
			t.Fatalf("expected legacy %s without If-Match to return 400 InvalidInput, got %d body=%s", tt.method, resp.StatusCode, string(resp.RawBody))
		}
	}

	unchanged, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
	if err != nil {
		t.Fatalf("get entity after rejected legacy mutations returned error: %v", err)
	}
	current := decodeJSONBody(t, unchanged)
	if current["Value"] != "original" || current["Score"] != float64(10) {
		t.Fatalf("expected rejected legacy mutations not to change entity, got %v", current)
	}

	modernPut, err := svc.HandleRequest(storageCtx(t, http.MethodPut, entityURL, []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"modern-put","Score":40}`), headers))
	if err != nil {
		t.Fatalf("modern put without If-Match returned error: %v", err)
	}
	if modernPut.StatusCode != http.StatusNoContent {
		t.Fatalf("expected modern put without If-Match to upsert with 204, got %d body=%s", modernPut.StatusCode, string(modernPut.RawBody))
	}
}

func TestTableDataPlaneDeleteEntityRequiresIfMatch(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"keep"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}

	entityURL := baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')"
	missingIfMatch, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, entityURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete without If-Match returned error: %v", err)
	}
	if missingIfMatch.StatusCode != http.StatusBadRequest || !strings.Contains(string(missingIfMatch.RawBody), "InvalidInput") {
		t.Fatalf("expected delete without If-Match to return 400 InvalidInput, got %d body=%s", missingIfMatch.StatusCode, string(missingIfMatch.RawBody))
	}

	stillExists, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
	if err != nil {
		t.Fatalf("get entity after rejected delete returned error: %v", err)
	}
	if stillExists.StatusCode != http.StatusOK {
		t.Fatalf("expected missing If-Match delete to preserve entity, got status %d body=%s", stillExists.StatusCode, string(stillExists.RawBody))
	}

	deleteWithWildcard, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, entityURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"If-Match":     "*",
	}))
	if err != nil {
		t.Fatalf("delete with wildcard If-Match returned error: %v", err)
	}
	if deleteWithWildcard.StatusCode != http.StatusNoContent {
		t.Fatalf("expected wildcard If-Match delete to return 204, got %d body=%s", deleteWithWildcard.StatusCode, string(deleteWithWildcard.RawBody))
	}
}

func TestTableDataPlaneInsertEntityHonorsReturnNoContentPreference(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
		"Prefer":       "return-no-content",
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"silent"}`), insertHeaders))
	if err != nil {
		t.Fatalf("insert entity with return-no-content returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected insert with return-no-content status 204, got %d body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	if len(insertEntity.RawBody) != 0 {
		t.Fatalf("expected insert with return-no-content to omit response body, got %s", string(insertEntity.RawBody))
	}
	if insertEntity.Headers["Preference-Applied"] != "return-no-content" {
		t.Fatalf("expected Preference-Applied return-no-content, got headers=%v", insertEntity.Headers)
	}
	if insertEntity.Headers["ETag"] == "" {
		t.Fatalf("expected insert with return-no-content to include ETag, got headers=%v", insertEntity.Headers)
	}
}

func TestTableDataPlaneInsertEntityHonorsReturnContentPreference(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
		"Prefer":       "return-content",
	}
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"echo"}`), insertHeaders))
	if err != nil {
		t.Fatalf("insert entity with return-content returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert with return-content status 201, got %d body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	if insertEntity.Headers["Preference-Applied"] != "return-content" {
		t.Fatalf("expected Preference-Applied return-content, got headers=%v", insertEntity.Headers)
	}
	inserted := decodeJSONBody(t, insertEntity)
	if inserted["PartitionKey"] != "p1" || inserted["RowKey"] != "r1" || inserted["Value"] != "echo" {
		t.Fatalf("expected echoed entity content, got %v", inserted)
	}
}

func TestTableDataPlaneInsertEntityWithoutPreferDoesNotSetPreferenceApplied(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"default"}`), headers))
	if err != nil {
		t.Fatalf("insert entity without Prefer returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert without Prefer status 201, got %d body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	if insertEntity.Headers["Preference-Applied"] != "" {
		t.Fatalf("expected no Preference-Applied header without Prefer, got headers=%v", insertEntity.Headers)
	}
}

func TestTableDataPlaneDoesNotPersistNullProperties(t *testing.T) {
	svc := storage.New()
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}
	baseURL := "http://localhost:4577/devstoreaccount1-table"

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	entityURL := baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')"
	insertEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tasks", []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":"initial","Optional":null,"Keep":"yes"}`), headers))
	if err != nil {
		t.Fatalf("insert entity returned error: %v", err)
	}
	if insertEntity.StatusCode != http.StatusCreated {
		t.Fatalf("expected insert entity status 201, got %d; body=%s", insertEntity.StatusCode, string(insertEntity.RawBody))
	}
	inserted := decodeJSONBody(t, insertEntity)
	if _, ok := inserted["Optional"]; ok {
		t.Fatalf("expected insert to omit null Optional property, got %v", inserted)
	}

	putHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	replaceEntity, err := svc.HandleRequest(storageCtx(t, http.MethodPut, entityURL, []byte(`{"PartitionKey":"p1","RowKey":"r1","Value":null,"Keep":"still"}`), putHeaders))
	if err != nil {
		t.Fatalf("replace entity returned error: %v", err)
	}
	if replaceEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected replace entity status 204, got %d; body=%s", replaceEntity.StatusCode, string(replaceEntity.RawBody))
	}
	replacedResp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
	if err != nil {
		t.Fatalf("get replaced entity returned error: %v", err)
	}
	replaced := decodeJSONBody(t, replacedResp)
	if _, ok := replaced["Value"]; ok {
		t.Fatalf("expected replace with null Value to omit property, got %v", replaced)
	}
	if replaced["Keep"] != "still" {
		t.Fatalf("expected replace to keep non-null property, got %v", replaced)
	}

	mergeHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "application/json",
		"If-Match":     "*",
	}
	mergeEntity, err := svc.HandleRequest(storageCtx(t, "MERGE", entityURL, []byte(`{"Keep":null,"Value":"merged"}`), mergeHeaders))
	if err != nil {
		t.Fatalf("merge entity returned error: %v", err)
	}
	if mergeEntity.StatusCode != http.StatusNoContent {
		t.Fatalf("expected merge entity status 204, got %d; body=%s", mergeEntity.StatusCode, string(mergeEntity.RawBody))
	}
	mergedResp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
	if err != nil {
		t.Fatalf("get merged entity returned error: %v", err)
	}
	merged := decodeJSONBody(t, mergedResp)
	if merged["Keep"] != "still" || merged["Value"] != "merged" {
		t.Fatalf("expected merge null to preserve Keep and set non-null Value, got %v", merged)
	}
}

func TestTableDataPlaneBatchInsertChangeset(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-table"
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	batchBody := strings.Join([]string{
		"--batch_1",
		"Content-Type: multipart/mixed; boundary=changeset_1",
		"",
		"--changeset_1",
		"Content-Type: application/http",
		"Content-Transfer-Encoding: binary",
		"Content-ID: 1",
		"",
		"POST " + baseURL + "/Tasks HTTP/1.1",
		"Content-Type: application/json",
		"Accept: application/json;odata=nometadata",
		"Prefer: return-no-content",
		"",
		`{"PartitionKey":"p1","RowKey":"r1","Value":"from-batch"}`,
		"--changeset_1--",
		"--batch_1--",
		"",
	}, "\r\n")
	batchHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "multipart/mixed; boundary=batch_1",
	}
	batchInsert, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/$batch", []byte(batchBody), batchHeaders))
	if err != nil {
		t.Fatalf("batch insert returned error: %v", err)
	}
	if batchInsert.StatusCode != http.StatusAccepted {
		t.Fatalf("expected batch insert status 202, got %d; body=%s", batchInsert.StatusCode, string(batchInsert.RawBody))
	}
	if !strings.HasPrefix(batchInsert.RawContentType, "multipart/mixed; boundary=batchresponse_") {
		t.Fatalf("expected multipart batch response content type, got %q", batchInsert.RawContentType)
	}
	batchResponse := string(batchInsert.RawBody)
	if !strings.Contains(batchResponse, "HTTP/1.1 204 No Content") || !strings.Contains(batchResponse, "ETag:") {
		t.Fatalf("expected batch response to contain operation status and ETag, got: %s", batchResponse)
	}

	getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/Tasks(PartitionKey='p1',RowKey='r1')", nil, headers))
	if err != nil {
		t.Fatalf("get batch entity returned error: %v", err)
	}
	inserted := decodeJSONBody(t, getEntity)
	if inserted["Value"] != "from-batch" {
		t.Fatalf("expected batch insert to create readable entity, got %v", inserted)
	}
}

func TestTableDataPlaneBatchRejectsMixedPartitionsAtomically(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-table"
	headers := map[string]string{
		"x-ms-version": "2023-11-03",
		"Accept":       "application/json;odata=nometadata",
		"Content-Type": "application/json",
	}

	createTable, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/Tables", []byte(`{"TableName":"Tasks"}`), headers))
	if err != nil {
		t.Fatalf("create table returned error: %v", err)
	}
	if createTable.StatusCode != http.StatusCreated {
		t.Fatalf("expected create table status 201, got %d; body=%s", createTable.StatusCode, string(createTable.RawBody))
	}

	batchBody := strings.Join([]string{
		"--batch_2",
		"Content-Type: multipart/mixed; boundary=changeset_2",
		"",
		"--changeset_2",
		"Content-Type: application/http",
		"Content-Transfer-Encoding: binary",
		"Content-ID: 1",
		"",
		"POST " + baseURL + "/Tasks HTTP/1.1",
		"Content-Type: application/json",
		"Prefer: return-no-content",
		"",
		`{"PartitionKey":"p1","RowKey":"r1","Value":"first"}`,
		"--changeset_2",
		"Content-Type: application/http",
		"Content-Transfer-Encoding: binary",
		"Content-ID: 2",
		"",
		"POST " + baseURL + "/Tasks HTTP/1.1",
		"Content-Type: application/json",
		"Prefer: return-no-content",
		"",
		`{"PartitionKey":"p2","RowKey":"r2","Value":"second"}`,
		"--changeset_2--",
		"--batch_2--",
		"",
	}, "\r\n")
	batchHeaders := map[string]string{
		"x-ms-version": "2023-11-03",
		"Content-Type": "multipart/mixed; boundary=batch_2",
	}
	mixedPartitionBatch, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/$batch", []byte(batchBody), batchHeaders))
	if err != nil {
		t.Fatalf("mixed partition batch returned error: %v", err)
	}
	if mixedPartitionBatch.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected mixed partition batch status 400, got %d; body=%s", mixedPartitionBatch.StatusCode, string(mixedPartitionBatch.RawBody))
	}

	for _, entityURL := range []string{
		baseURL + "/Tasks(PartitionKey='p1',RowKey='r1')",
		baseURL + "/Tasks(PartitionKey='p2',RowKey='r2')",
	} {
		getEntity, err := svc.HandleRequest(storageCtx(t, http.MethodGet, entityURL, nil, headers))
		if err != nil {
			t.Fatalf("get entity after rejected batch returned error: %v", err)
		}
		if getEntity.StatusCode != http.StatusNotFound {
			t.Fatalf("expected rejected batch not to create %s, got %d body=%s", entityURL, getEntity.StatusCode, string(getEntity.RawBody))
		}
	}
}

func TestBlobListRejectsInvalidMaxResults(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	listBlobs, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs?restype=container&comp=list&maxresults=0", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list blobs returned error: %v", err)
	}
	if listBlobs.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid blob maxresults status 400, got %d; body=%s", listBlobs.StatusCode, string(listBlobs.RawBody))
	}
}

func TestBlobDataPlaneGetAccountInformationReturnsSkuAndKindForAnyPath(t *testing.T) {
	svc := storage.New()
	_, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/acctest?api-version=2024-01-01", []byte(`{"location":"westus2","kind":"BlobStorage","sku":{"name":"Premium_LRS"}}`), nil))
	if err != nil {
		t.Fatalf("create storage account returned error: %v", err)
	}

	headInfo, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "https://acctest.blob.core.windows.net/?restype=account&comp=properties", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "account-info-1",
	}))
	if err != nil {
		t.Fatalf("head account information returned error: %v", err)
	}
	if headInfo.StatusCode != http.StatusOK || len(headInfo.RawBody) != 0 {
		t.Fatalf("expected account information HEAD status 200 without body, got status=%d body=%q", headInfo.StatusCode, string(headInfo.RawBody))
	}
	if headInfo.Headers["Content-Length"] != "0" ||
		headInfo.Headers["x-ms-sku-name"] != "Premium_LRS" ||
		headInfo.Headers["x-ms-account-kind"] != "BlobStorage" ||
		headInfo.Headers["x-ms-is-hns-enabled"] != "false" ||
		headInfo.Headers["x-ms-client-request-id"] != "account-info-1" {
		t.Fatalf("expected account information headers from stored account, got %v", headInfo.Headers)
	}

	getInfo, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/missing/container/blob.txt?restype=account&comp=properties", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get local account information returned error: %v", err)
	}
	if getInfo.StatusCode != http.StatusOK || len(getInfo.RawBody) != 0 {
		t.Fatalf("expected local account information GET status 200 without body, got status=%d body=%q", getInfo.StatusCode, string(getInfo.RawBody))
	}
	if getInfo.Headers["Content-Length"] != "0" ||
		getInfo.Headers["x-ms-sku-name"] != "Standard_LRS" ||
		getInfo.Headers["x-ms-account-kind"] != "StorageV2" ||
		getInfo.Headers["x-ms-is-hns-enabled"] != "false" {
		t.Fatalf("expected default local account information headers, got %v", getInfo.Headers)
	}
}

func TestBlobDataPlaneHeadReturnsPropertiesWithoutBody(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/readme.txt", []byte("hello azure"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "docs",
	}))

	headBlob, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head blob returned error: %v", err)
	}
	if headBlob.StatusCode != http.StatusOK {
		t.Fatalf("expected head blob status 200, got %d; body=%s", headBlob.StatusCode, string(headBlob.RawBody))
	}
	if len(headBlob.RawBody) != 0 ||
		headBlob.Headers["Content-Length"] != "11" ||
		headBlob.Headers["Content-Type"] != "text/plain" ||
		headBlob.Headers["x-ms-meta-owner"] != "docs" ||
		headBlob.Headers["x-ms-blob-type"] != "BlockBlob" ||
		headBlob.Headers["Accept-Ranges"] != "bytes" ||
		headBlob.Headers["ETag"] == "" ||
		headBlob.Headers["Last-Modified"] == "" {
		t.Fatalf("expected blob property headers without body, headers=%v body=%q", headBlob.Headers, string(headBlob.RawBody))
	}
}

func TestBlobDataPlaneAppendBlobInitializationAndAppendBlock(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs?restype=container", nil, nil))

	createAppendBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log", nil, map[string]string{
		"x-ms-blob-type":  "AppendBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "observability",
	}))
	if err != nil {
		t.Fatalf("create append blob returned error: %v", err)
	}
	if createAppendBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected create append blob status 201, got %d body=%s", createAppendBlob.StatusCode, string(createAppendBlob.RawBody))
	}

	headEmpty, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/logs/current.log", nil, nil))
	if err != nil {
		t.Fatalf("head empty append blob returned error: %v", err)
	}
	if headEmpty.Headers["x-ms-blob-type"] != "AppendBlob" ||
		headEmpty.Headers["Content-Length"] != "0" ||
		headEmpty.Headers["x-ms-blob-committed-block-count"] != "0" ||
		headEmpty.Headers["x-ms-meta-owner"] != "observability" {
		t.Fatalf("expected initialized append blob properties, got %v", headEmpty.Headers)
	}

	firstAppend, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log?comp=appendblock", []byte("alpha"), nil))
	if err != nil {
		t.Fatalf("first append block returned error: %v", err)
	}
	if firstAppend.StatusCode != http.StatusCreated ||
		firstAppend.Headers["x-ms-blob-append-offset"] != "0" ||
		firstAppend.Headers["x-ms-blob-committed-block-count"] != "1" {
		t.Fatalf("expected first append offset/count headers, status=%d headers=%v body=%s", firstAppend.StatusCode, firstAppend.Headers, string(firstAppend.RawBody))
	}

	secondAppend, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log?comp=appendblock", []byte(" beta"), map[string]string{
		"x-ms-blob-condition-appendpos": "5",
		"x-ms-blob-condition-maxsize":   "10",
	}))
	if err != nil {
		t.Fatalf("second append block returned error: %v", err)
	}
	if secondAppend.StatusCode != http.StatusCreated ||
		secondAppend.Headers["x-ms-blob-append-offset"] != "5" ||
		secondAppend.Headers["x-ms-blob-committed-block-count"] != "2" {
		t.Fatalf("expected second append offset/count headers, status=%d headers=%v body=%s", secondAppend.StatusCode, secondAppend.Headers, string(secondAppend.RawBody))
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/logs/current.log", nil, nil))
	if err != nil {
		t.Fatalf("get append blob returned error: %v", err)
	}
	if string(getBlob.RawBody) != "alpha beta" ||
		getBlob.Headers["x-ms-blob-type"] != "AppendBlob" ||
		getBlob.Headers["x-ms-blob-committed-block-count"] != "2" ||
		getBlob.RawContentType != "text/plain" {
		t.Fatalf("expected appended content and append blob headers, contentType=%q headers=%v body=%q", getBlob.RawContentType, getBlob.Headers, string(getBlob.RawBody))
	}
}

func TestBlobDataPlaneAppendBlockValidatesTargetTypeAndConditions(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/block.txt", []byte("block"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	appendMissing, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/missing.log?comp=appendblock", []byte("alpha"), nil))
	if err != nil {
		t.Fatalf("append missing blob returned error: %v", err)
	}
	if appendMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected append block against missing blob to fail 404, got %d body=%s", appendMissing.StatusCode, string(appendMissing.RawBody))
	}

	appendBlockBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/block.txt?comp=appendblock", []byte("alpha"), nil))
	if err != nil {
		t.Fatalf("append block blob returned error: %v", err)
	}
	if appendBlockBlob.StatusCode != http.StatusConflict {
		t.Fatalf("expected append block against block blob to fail 409, got %d body=%s", appendBlockBlob.StatusCode, string(appendBlockBlob.RawBody))
	}

	createWithBody, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/bad.log", []byte("not-empty"), map[string]string{
		"x-ms-blob-type": "AppendBlob",
	}))
	if err != nil {
		t.Fatalf("create append blob with body returned error: %v", err)
	}
	if createWithBody.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected AppendBlob Put Blob with body to fail 400, got %d body=%s", createWithBody.StatusCode, string(createWithBody.RawBody))
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log", nil, map[string]string{
		"x-ms-blob-type": "AppendBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log?comp=appendblock", []byte("alpha"), nil))

	badAppendPos, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log?comp=appendblock", []byte(" beta"), map[string]string{
		"x-ms-blob-condition-appendpos": "99",
	}))
	if err != nil {
		t.Fatalf("append with bad append position returned error: %v", err)
	}
	if badAppendPos.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected append position mismatch to fail 412, got %d body=%s", badAppendPos.StatusCode, string(badAppendPos.RawBody))
	}

	badMaxSize, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/current.log?comp=appendblock", []byte(" beta"), map[string]string{
		"x-ms-blob-condition-maxsize": "6",
	}))
	if err != nil {
		t.Fatalf("append with max size condition returned error: %v", err)
	}
	if badMaxSize.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected max size condition to fail 412, got %d body=%s", badMaxSize.StatusCode, string(badMaxSize.RawBody))
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/logs/current.log", nil, nil))
	if err != nil {
		t.Fatalf("get append blob after failed appends returned error: %v", err)
	}
	if string(getBlob.RawBody) != "alpha" || getBlob.Headers["x-ms-blob-committed-block-count"] != "1" {
		t.Fatalf("expected failed append conditions not to mutate blob, headers=%v body=%q", getBlob.Headers, string(getBlob.RawBody))
	}
}

func TestBlobDataPlanePageBlobPutPageAndPageList(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks?restype=container", nil, nil))

	createPageBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, map[string]string{
		"x-ms-blob-type":            "PageBlob",
		"x-ms-blob-content-length":  "1024",
		"x-ms-blob-sequence-number": "7",
		"Content-Type":              "application/octet-stream",
	}))
	if err != nil {
		t.Fatalf("create page blob returned error: %v", err)
	}
	if createPageBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected create page blob status 201, got %d body=%s", createPageBlob.StatusCode, string(createPageBlob.RawBody))
	}

	headEmpty, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("head empty page blob returned error: %v", err)
	}
	if headEmpty.Headers["x-ms-blob-type"] != "PageBlob" ||
		headEmpty.Headers["Content-Length"] != "1024" ||
		headEmpty.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected initialized page blob headers, got %v", headEmpty.Headers)
	}

	firstPage := []byte(strings.Repeat("A", 512))
	putFirstPage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", firstPage, map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=0-511",
	}))
	if err != nil {
		t.Fatalf("put first page returned error: %v", err)
	}
	if putFirstPage.StatusCode != http.StatusCreated || putFirstPage.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected first page update 201 with sequence number, got status=%d headers=%v body=%s", putFirstPage.StatusCode, putFirstPage.Headers, string(putFirstPage.RawBody))
	}

	secondPage := []byte(strings.Repeat("B", 512))
	putSecondPage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", secondPage, map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=512-1023",
	}))
	if err != nil {
		t.Fatalf("put second page returned error: %v", err)
	}
	if putSecondPage.StatusCode != http.StatusCreated {
		t.Fatalf("expected second page update 201, got %d body=%s", putSecondPage.StatusCode, string(putSecondPage.RawBody))
	}

	getFull, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get page blob returned error: %v", err)
	}
	if !bytes.Equal(getFull.RawBody, append(append([]byte{}, firstPage...), secondPage...)) ||
		getFull.Headers["x-ms-blob-type"] != "PageBlob" ||
		getFull.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected two updated pages and page blob headers, headers=%v body-prefix=%q", getFull.Headers, string(getFull.RawBody[:16]))
	}

	clearSecondPage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", nil, map[string]string{
		"x-ms-page-write": "clear",
		"Range":           "bytes=512-1023",
	}))
	if err != nil {
		t.Fatalf("clear second page returned error: %v", err)
	}
	if clearSecondPage.StatusCode != http.StatusCreated {
		t.Fatalf("expected clear page status 201, got %d body=%s", clearSecondPage.StatusCode, string(clearSecondPage.RawBody))
	}

	getAfterClear, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get page blob after clear returned error: %v", err)
	}
	expectedAfterClear := append(append([]byte{}, firstPage...), make([]byte, 512)...)
	if !bytes.Equal(getAfterClear.RawBody, expectedAfterClear) {
		t.Fatalf("expected cleared page to read as zero bytes")
	}

	pageList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=pagelist", nil, nil))
	if err != nil {
		t.Fatalf("get page ranges returned error: %v", err)
	}
	pageListBody := string(pageList.RawBody)
	if pageList.StatusCode != http.StatusOK ||
		!strings.Contains(pageListBody, "<PageList>") ||
		!strings.Contains(pageListBody, "<Start>0</Start>") ||
		!strings.Contains(pageListBody, "<End>511</End>") ||
		strings.Contains(pageListBody, "<Start>512</Start>") {
		t.Fatalf("expected page list with only first allocated range, status=%d body=%s", pageList.StatusCode, pageListBody)
	}
}

func TestBlobDataPlanePageBlobValidatesTypeRangeAndLength(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/block.bin", []byte("block"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	createWithBody, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/body.vhd", []byte("not-empty"), map[string]string{
		"x-ms-blob-type":           "PageBlob",
		"x-ms-blob-content-length": "1024",
	}))
	if err != nil {
		t.Fatalf("create page blob with body returned error: %v", err)
	}
	if createWithBody.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected PageBlob Put Blob with body to fail 400, got %d body=%s", createWithBody.StatusCode, string(createWithBody.RawBody))
	}

	createUnaligned, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/unaligned.vhd", nil, map[string]string{
		"x-ms-blob-type":           "PageBlob",
		"x-ms-blob-content-length": "513",
	}))
	if err != nil {
		t.Fatalf("create unaligned page blob returned error: %v", err)
	}
	if createUnaligned.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unaligned page blob length to fail 400, got %d body=%s", createUnaligned.StatusCode, string(createUnaligned.RawBody))
	}

	putPageMissing, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/missing.vhd?comp=page", []byte(strings.Repeat("A", 512)), map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=0-511",
	}))
	if err != nil {
		t.Fatalf("put page missing blob returned error: %v", err)
	}
	if putPageMissing.StatusCode != http.StatusNotFound {
		t.Fatalf("expected put page against missing blob to fail 404, got %d body=%s", putPageMissing.StatusCode, string(putPageMissing.RawBody))
	}

	putPageBlockBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/block.bin?comp=page", []byte(strings.Repeat("A", 512)), map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=0-511",
	}))
	if err != nil {
		t.Fatalf("put page block blob returned error: %v", err)
	}
	if putPageBlockBlob.StatusCode != http.StatusConflict {
		t.Fatalf("expected put page against block blob to fail 409, got %d body=%s", putPageBlockBlob.StatusCode, string(putPageBlockBlob.RawBody))
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, map[string]string{
		"x-ms-blob-type":           "PageBlob",
		"x-ms-blob-content-length": "1024",
	}))

	putUnalignedPage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", []byte(strings.Repeat("A", 512)), map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=1-512",
	}))
	if err != nil {
		t.Fatalf("put unaligned page returned error: %v", err)
	}
	if putUnalignedPage.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unaligned page range to fail 400, got %d body=%s", putUnalignedPage.StatusCode, string(putUnalignedPage.RawBody))
	}

	putMismatchedLength, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", []byte("short"), map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=0-511",
	}))
	if err != nil {
		t.Fatalf("put mismatched page length returned error: %v", err)
	}
	if putMismatchedLength.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected mismatched page length to fail 400, got %d body=%s", putMismatchedLength.StatusCode, string(putMismatchedLength.RawBody))
	}

	getPageBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get page blob after failed writes returned error: %v", err)
	}
	if !bytes.Equal(getPageBlob.RawBody, make([]byte, 1024)) {
		t.Fatalf("expected failed page writes not to mutate blob")
	}
}

func TestBlobDataPlanePageBlobPutPageSequenceNumberConditions(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, map[string]string{
		"x-ms-blob-type":            "PageBlob",
		"x-ms-blob-content-length":  "1024",
		"x-ms-blob-sequence-number": "7",
		"Content-Type":              "application/octet-stream",
	}))

	firstPage := []byte(strings.Repeat("A", 512))
	putEqual, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", firstPage, map[string]string{
		"x-ms-page-write":            "update",
		"x-ms-range":                 "bytes=0-511",
		"x-ms-if-sequence-number-eq": "7",
	}))
	if err != nil {
		t.Fatalf("put page with eq sequence condition returned error: %v", err)
	}
	if putEqual.StatusCode != http.StatusCreated || putEqual.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected equal sequence condition to succeed, status=%d headers=%v body=%s", putEqual.StatusCode, putEqual.Headers, string(putEqual.RawBody))
	}

	secondPage := []byte(strings.Repeat("B", 512))
	putLessThan, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", secondPage, map[string]string{
		"x-ms-page-write":            "update",
		"x-ms-range":                 "bytes=512-1023",
		"x-ms-if-sequence-number-le": "7",
		"x-ms-if-sequence-number-lt": "8",
	}))
	if err != nil {
		t.Fatalf("put page with le/lt sequence conditions returned error: %v", err)
	}
	if putLessThan.StatusCode != http.StatusCreated || putLessThan.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected le/lt sequence conditions to succeed, status=%d headers=%v body=%s", putLessThan.StatusCode, putLessThan.Headers, string(putLessThan.RawBody))
	}

	failingCases := []struct {
		name   string
		header string
		value  string
	}{
		{name: "eq", header: "x-ms-if-sequence-number-eq", value: "6"},
		{name: "le", header: "x-ms-if-sequence-number-le", value: "6"},
		{name: "lt", header: "x-ms-if-sequence-number-lt", value: "7"},
	}
	for _, tc := range failingCases {
		t.Run(tc.name, func(t *testing.T) {
			headers := map[string]string{
				"x-ms-page-write": "update",
				"x-ms-range":      "bytes=0-511",
				tc.header:         tc.value,
			}
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", []byte(strings.Repeat("Z", 512)), headers))
			if err != nil {
				t.Fatalf("put page with failing sequence condition returned error: %v", err)
			}
			if resp.StatusCode != http.StatusPreconditionFailed {
				t.Fatalf("expected failing %s sequence condition to return 412, got %d body=%s", tc.name, resp.StatusCode, string(resp.RawBody))
			}
		})
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get page blob after failed sequence conditions returned error: %v", err)
	}
	expected := append(append([]byte{}, firstPage...), secondPage...)
	if !bytes.Equal(getBlob.RawBody, expected) || getBlob.Headers["x-ms-blob-sequence-number"] != "7" {
		t.Fatalf("expected failed sequence conditions not to mutate page blob, headers=%v", getBlob.Headers)
	}
}

func TestBlobDataPlaneSetPropertiesUpdatesPageBlobSequenceNumber(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, map[string]string{
		"x-ms-blob-type":            "PageBlob",
		"x-ms-blob-content-length":  "1024",
		"x-ms-blob-sequence-number": "7",
		"Content-Type":              "application/octet-stream",
	}))

	increment, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=properties", nil, map[string]string{
		"x-ms-sequence-number-action": "increment",
	}))
	if err != nil {
		t.Fatalf("increment sequence number returned error: %v", err)
	}
	if increment.StatusCode != http.StatusOK || increment.Headers["x-ms-blob-sequence-number"] != "8" {
		t.Fatalf("expected increment to return sequence number 8, status=%d headers=%v body=%s", increment.StatusCode, increment.Headers, string(increment.RawBody))
	}

	update, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=properties", nil, map[string]string{
		"x-ms-sequence-number-action": "update",
		"x-ms-blob-sequence-number":   "3",
	}))
	if err != nil {
		t.Fatalf("update sequence number returned error: %v", err)
	}
	if update.StatusCode != http.StatusOK || update.Headers["x-ms-blob-sequence-number"] != "3" {
		t.Fatalf("expected update to return sequence number 3, status=%d headers=%v body=%s", update.StatusCode, update.Headers, string(update.RawBody))
	}

	maxHigher, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=properties", nil, map[string]string{
		"x-ms-sequence-number-action": "max",
		"x-ms-blob-sequence-number":   "9",
	}))
	if err != nil {
		t.Fatalf("max higher sequence number returned error: %v", err)
	}
	if maxHigher.StatusCode != http.StatusOK || maxHigher.Headers["x-ms-blob-sequence-number"] != "9" {
		t.Fatalf("expected max higher to return sequence number 9, status=%d headers=%v body=%s", maxHigher.StatusCode, maxHigher.Headers, string(maxHigher.RawBody))
	}

	maxLower, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=properties", nil, map[string]string{
		"x-ms-sequence-number-action": "max",
		"x-ms-blob-sequence-number":   "4",
	}))
	if err != nil {
		t.Fatalf("max lower sequence number returned error: %v", err)
	}
	if maxLower.StatusCode != http.StatusOK || maxLower.Headers["x-ms-blob-sequence-number"] != "9" {
		t.Fatalf("expected max lower to keep sequence number 9, status=%d headers=%v body=%s", maxLower.StatusCode, maxLower.Headers, string(maxLower.RawBody))
	}

	headBlob, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/disks/os.vhd", nil, nil))
	if err != nil {
		t.Fatalf("head page blob returned error: %v", err)
	}
	if headBlob.Headers["x-ms-blob-sequence-number"] != "9" {
		t.Fatalf("expected head blob to project sequence number 9, got headers=%v", headBlob.Headers)
	}
	if headBlob.Headers["Content-Type"] != "application/octet-stream" {
		t.Fatalf("expected sequence-only property updates to preserve content type, got headers=%v", headBlob.Headers)
	}

	putPage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/os.vhd?comp=page", []byte(strings.Repeat("A", 512)), map[string]string{
		"x-ms-page-write":            "update",
		"x-ms-range":                 "bytes=0-511",
		"x-ms-if-sequence-number-eq": "9",
	}))
	if err != nil {
		t.Fatalf("put page with updated sequence condition returned error: %v", err)
	}
	if putPage.StatusCode != http.StatusCreated {
		t.Fatalf("expected put page with updated sequence condition to succeed, got %d body=%s", putPage.StatusCode, string(putPage.RawBody))
	}
}

func TestBlobDataPlaneSetPropertiesResizesPageBlob(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/data.vhd", nil, map[string]string{
		"x-ms-blob-type":            "PageBlob",
		"x-ms-blob-content-length":  "1536",
		"x-ms-blob-sequence-number": "5",
		"Content-Type":              "application/octet-stream",
	}))
	firstPage := []byte(strings.Repeat("A", 512))
	lastPage := []byte(strings.Repeat("C", 512))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/data.vhd?comp=page", firstPage, map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=0-511",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/data.vhd?comp=page", lastPage, map[string]string{
		"x-ms-page-write": "update",
		"x-ms-range":      "bytes=1024-1535",
	}))

	shrink, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/data.vhd?comp=properties", nil, map[string]string{
		"x-ms-blob-content-length": "1024",
	}))
	if err != nil {
		t.Fatalf("shrink page blob returned error: %v", err)
	}
	if shrink.StatusCode != http.StatusOK || shrink.Headers["x-ms-blob-sequence-number"] != "5" {
		t.Fatalf("expected shrink to return status 200 and preserve sequence number 5, status=%d headers=%v body=%s", shrink.StatusCode, shrink.Headers, string(shrink.RawBody))
	}

	headShrunk, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/disks/data.vhd", nil, nil))
	if err != nil {
		t.Fatalf("head shrunk page blob returned error: %v", err)
	}
	if headShrunk.Headers["Content-Length"] != "1024" ||
		headShrunk.Headers["x-ms-blob-sequence-number"] != "5" ||
		headShrunk.Headers["Content-Type"] != "application/octet-stream" {
		t.Fatalf("expected shrunk page blob headers to preserve properties, got %v", headShrunk.Headers)
	}

	getShrunk, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/data.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get shrunk page blob returned error: %v", err)
	}
	expectedShrunk := append(append([]byte{}, firstPage...), make([]byte, 512)...)
	if !bytes.Equal(getShrunk.RawBody, expectedShrunk) {
		t.Fatalf("expected shrink to truncate out-of-range page data, length=%d suffix=%q", len(getShrunk.RawBody), string(getShrunk.RawBody[512:]))
	}

	pageList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/data.vhd?comp=pagelist", nil, nil))
	if err != nil {
		t.Fatalf("get page ranges after shrink returned error: %v", err)
	}
	pageListBody := string(pageList.RawBody)
	if !strings.Contains(pageListBody, "<Start>0</Start>") ||
		!strings.Contains(pageListBody, "<End>511</End>") ||
		strings.Contains(pageListBody, "<Start>1024</Start>") {
		t.Fatalf("expected shrink to clear page ranges outside the new length, body=%s", pageListBody)
	}

	grow, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/disks/data.vhd?comp=properties", nil, map[string]string{
		"x-ms-blob-content-length": "2048",
	}))
	if err != nil {
		t.Fatalf("grow page blob returned error: %v", err)
	}
	if grow.StatusCode != http.StatusOK || grow.Headers["x-ms-blob-sequence-number"] != "5" {
		t.Fatalf("expected grow to return status 200 and preserve sequence number 5, status=%d headers=%v body=%s", grow.StatusCode, grow.Headers, string(grow.RawBody))
	}

	getGrown, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/disks/data.vhd", nil, nil))
	if err != nil {
		t.Fatalf("get grown page blob returned error: %v", err)
	}
	expectedGrown := append(append([]byte{}, expectedShrunk...), make([]byte, 1024)...)
	if !bytes.Equal(getGrown.RawBody, expectedGrown) || getGrown.Headers["Content-Length"] != "2048" || getGrown.Headers["Content-Type"] != "application/octet-stream" {
		t.Fatalf("expected grow to extend with zeroes and preserve properties, headers=%v length=%d", getGrown.Headers, len(getGrown.RawBody))
	}
}

func TestBlobDataPlaneSetPropertiesUpdatesHeadersAndPreservesContent(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/readme.txt", []byte("hello azure"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "docs",
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d; body=%s", putBlob.StatusCode, string(putBlob.RawBody))
	}
	oldETag := putBlob.Headers["ETag"]

	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/readme.txt?comp=properties", nil, map[string]string{
		"x-ms-blob-content-type":        "application/json",
		"x-ms-blob-cache-control":       "max-age=60",
		"x-ms-blob-content-language":    "en-US",
		"x-ms-blob-content-encoding":    "gzip",
		"x-ms-blob-content-disposition": `attachment; filename="readme.txt"`,
	}))
	if err != nil {
		t.Fatalf("set blob properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusOK {
		t.Fatalf("expected set properties status 200, got %d; body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}
	if setProperties.Headers["ETag"] == "" || setProperties.Headers["ETag"] == oldETag || len(setProperties.RawBody) != 0 {
		t.Fatalf("expected changed ETag and no body, old=%q headers=%v body=%q", oldETag, setProperties.Headers, string(setProperties.RawBody))
	}

	headBlob, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/readme.txt", nil, nil))
	if err != nil {
		t.Fatalf("head blob returned error: %v", err)
	}
	if headBlob.Headers["Content-Type"] != "application/json" ||
		headBlob.Headers["Cache-Control"] != "max-age=60" ||
		headBlob.Headers["Content-Language"] != "en-US" ||
		headBlob.Headers["Content-Encoding"] != "gzip" ||
		headBlob.Headers["Content-Disposition"] != `attachment; filename="readme.txt"` ||
		headBlob.Headers["x-ms-meta-owner"] != "docs" ||
		headBlob.Headers["Content-Length"] != "11" {
		t.Fatalf("expected updated blob property headers and preserved metadata, got %v", headBlob.Headers)
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/readme.txt", nil, nil))
	if err != nil {
		t.Fatalf("get blob returned error: %v", err)
	}
	if string(getBlob.RawBody) != "hello azure" || getBlob.RawContentType != "application/json" {
		t.Fatalf("expected content to be preserved with updated content type, contentType=%q body=%q", getBlob.RawContentType, string(getBlob.RawBody))
	}
}

func TestBlobDataPlaneSetBlobTierPreservesETagAndProjectsProperties(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", []byte("tiered content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", nil, nil))
	if err != nil {
		t.Fatalf("initial head returned error: %v", err)
	}
	if headBefore.StatusCode != http.StatusOK || headBefore.Headers["ETag"] == "" || headBefore.Headers["Last-Modified"] == "" {
		t.Fatalf("expected initial properties with ETag and last-modified, got status=%d headers=%v", headBefore.StatusCode, headBefore.Headers)
	}

	setTier, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Cool",
	}))
	if err != nil {
		t.Fatalf("set blob tier returned error: %v", err)
	}
	if setTier.StatusCode != http.StatusOK {
		t.Fatalf("expected Set Blob Tier status 200, got %d body=%s", setTier.StatusCode, string(setTier.RawBody))
	}
	if setTier.Headers["ETag"] != headBefore.Headers["ETag"] || setTier.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected Set Blob Tier to preserve ETag and last-modified, before=%v after=%v", headBefore.Headers, setTier.Headers)
	}

	headAfter, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", nil, nil))
	if err != nil {
		t.Fatalf("head after set tier returned error: %v", err)
	}
	if headAfter.Headers["ETag"] != headBefore.Headers["ETag"] || headAfter.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected blob properties to preserve ETag and last-modified after tier change, before=%v after=%v", headBefore.Headers, headAfter.Headers)
	}
	if headAfter.Headers["x-ms-access-tier"] != "Cool" {
		t.Fatalf("expected access tier Cool in blob properties, got headers=%v", headAfter.Headers)
	}
	if headAfter.Headers["x-ms-access-tier-change-time"] == "" {
		t.Fatalf("expected access tier change time in blob properties, got headers=%v", headAfter.Headers)
	}

	getAfter, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", nil, nil))
	if err != nil {
		t.Fatalf("get after set tier returned error: %v", err)
	}
	if getAfter.StatusCode != http.StatusOK || string(getAfter.RawBody) != "tiered content" || getAfter.Headers["x-ms-access-tier"] != "Cool" {
		t.Fatalf("expected content and access tier to be preserved after set tier, status=%d headers=%v body=%q", getAfter.StatusCode, getAfter.Headers, string(getAfter.RawBody))
	}
}

func TestBlobDataPlaneSetBlobTierRehydratesArchiveBlobAsPending(t *testing.T) {
	svc := storage.New()
	blobURL := "http://localhost:4577/devstoreaccount1/docs/archive.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL, []byte("archived content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	archive, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Archive",
		"x-ms-version":     "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set archive tier returned error: %v", err)
	}
	if archive.StatusCode != http.StatusOK {
		t.Fatalf("expected archive tier set to return 200, got %d body=%s", archive.StatusCode, string(archive.RawBody))
	}

	headArchive, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head archived blob returned error: %v", err)
	}
	if headArchive.Headers["x-ms-access-tier"] != "Archive" || headArchive.Headers["x-ms-archive-status"] != "" {
		t.Fatalf("expected archived blob properties without pending rehydrate state, got %v", headArchive.Headers)
	}

	rehydrate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier":        "Cool",
		"x-ms-rehydrate-priority": "Standard",
		"x-ms-version":            "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("rehydrate archive blob returned error: %v", err)
	}
	if rehydrate.StatusCode != http.StatusAccepted {
		t.Fatalf("expected archive rehydrate to return 202, got %d body=%s", rehydrate.StatusCode, string(rehydrate.RawBody))
	}

	headPending, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head pending rehydrate blob returned error: %v", err)
	}
	if headPending.Headers["ETag"] != headArchive.Headers["ETag"] ||
		headPending.Headers["Last-Modified"] != headArchive.Headers["Last-Modified"] ||
		headPending.Headers["x-ms-access-tier"] != "Archive" ||
		headPending.Headers["x-ms-archive-status"] != "rehydrate-pending-to-cool" ||
		headPending.Headers["x-ms-rehydrate-priority"] != "Standard" {
		t.Fatalf("expected pending rehydrate headers with preserved blob validators, before=%v after=%v", headArchive.Headers, headPending.Headers)
	}

	raisePriority, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier":        "Cool",
		"x-ms-rehydrate-priority": "High",
		"x-ms-version":            "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("raise rehydrate priority returned error: %v", err)
	}
	if raisePriority.StatusCode != http.StatusAccepted {
		t.Fatalf("expected rehydrate priority update to return 202, got %d body=%s", raisePriority.StatusCode, string(raisePriority.RawBody))
	}

	retarget, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Hot",
		"x-ms-version":     "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("retarget pending rehydrate returned error: %v", err)
	}
	if retarget.StatusCode != http.StatusConflict {
		t.Fatalf("expected pending rehydrate retarget to return 409, got %d body=%s", retarget.StatusCode, string(retarget.RawBody))
	}

	headAfterRetarget, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head after rejected retarget returned error: %v", err)
	}
	if headAfterRetarget.Headers["x-ms-archive-status"] != "rehydrate-pending-to-cool" ||
		headAfterRetarget.Headers["x-ms-rehydrate-priority"] != "High" {
		t.Fatalf("expected rejected retarget to preserve pending cool rehydrate at High priority, got %v", headAfterRetarget.Headers)
	}
}

func TestBlobDataPlaneSetBlobTierRequiresAccessTierHeader(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", []byte("tiered content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Hot",
	}))

	missingTier, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?comp=tier", nil, nil))
	if err != nil {
		t.Fatalf("set blob tier without header returned error: %v", err)
	}
	if missingTier.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing access tier to return 400, got %d body=%s", missingTier.StatusCode, string(missingTier.RawBody))
	}

	headAfter, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", nil, nil))
	if err != nil {
		t.Fatalf("head after missing tier returned error: %v", err)
	}
	if headAfter.Headers["x-ms-access-tier"] != "Hot" {
		t.Fatalf("expected missing access tier request to preserve existing Hot tier, got headers=%v", headAfter.Headers)
	}
}

func TestBlobDataPlaneSetBlobTierRejectsInvalidAccessTier(t *testing.T) {
	svc := storage.New()
	blobURL := "http://localhost:4577/devstoreaccount1/docs/tiered.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL, []byte("tiered content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Hot",
		"x-ms-version":     "2023-11-03",
	}))

	invalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Glacier",
		"x-ms-version":     "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set invalid access tier returned error: %v", err)
	}
	if invalid.StatusCode != http.StatusBadRequest || invalid.Headers["x-ms-error-code"] != "InvalidHeaderValue" {
		t.Fatalf("expected invalid access tier to return 400 InvalidHeaderValue, got status=%d headers=%v body=%s", invalid.StatusCode, invalid.Headers, string(invalid.RawBody))
	}

	headAfter, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, nil))
	if err != nil {
		t.Fatalf("head after invalid access tier returned error: %v", err)
	}
	if headAfter.Headers["x-ms-access-tier"] != "Hot" {
		t.Fatalf("expected invalid access tier request to preserve Hot tier, got headers=%v", headAfter.Headers)
	}
}

func TestBlobDataPlaneSetBlobTierRejectsColdAndSmartBeforeSupportedVersions(t *testing.T) {
	svc := storage.New()
	blobURL := "http://localhost:4577/devstoreaccount1/docs/versioned-tier.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL, []byte("tiered content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Hot",
		"x-ms-version":     "2023-11-03",
	}))

	rejectedCases := []struct {
		name    string
		tier    string
		version string
	}{
		{name: "cold before 2021-12-02", tier: "Cold", version: "2020-10-02"},
		{name: "smart before 2026-02-06", tier: "Smart", version: "2025-11-05"},
	}
	for _, tc := range rejectedCases {
		t.Run(tc.name, func(t *testing.T) {
			rejected, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
				"x-ms-access-tier": tc.tier,
				"x-ms-version":     tc.version,
			}))
			if err != nil {
				t.Fatalf("set %s with version %s returned error: %v", tc.tier, tc.version, err)
			}
			if rejected.StatusCode != http.StatusBadRequest || rejected.Headers["x-ms-error-code"] != "FeatureVersionMismatch" {
				t.Fatalf("expected %s before supported version to return 400 FeatureVersionMismatch, got status=%d headers=%v body=%s", tc.tier, rejected.StatusCode, rejected.Headers, string(rejected.RawBody))
			}

			headAfter, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, nil))
			if err != nil {
				t.Fatalf("head after rejected %s tier returned error: %v", tc.tier, err)
			}
			if headAfter.Headers["x-ms-access-tier"] != "Hot" {
				t.Fatalf("expected rejected %s request to preserve Hot tier, got headers=%v", tc.tier, headAfter.Headers)
			}
		})
	}

	cold, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Cold",
		"x-ms-version":     "2021-12-02",
	}))
	if err != nil {
		t.Fatalf("set Cold at supported version returned error: %v", err)
	}
	if cold.StatusCode != http.StatusOK {
		t.Fatalf("expected Cold at supported version to return 200, got %d body=%s", cold.StatusCode, string(cold.RawBody))
	}

	smart, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier", nil, map[string]string{
		"x-ms-access-tier": "Smart",
		"x-ms-version":     "2026-02-06",
	}))
	if err != nil {
		t.Fatalf("set Smart at supported version returned error: %v", err)
	}
	if smart.StatusCode != http.StatusOK {
		t.Fatalf("expected Smart at supported version to return 200, got %d body=%s", smart.StatusCode, string(smart.RawBody))
	}
	headSmart, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL, nil, nil))
	if err != nil {
		t.Fatalf("head after supported Smart tier returned error: %v", err)
	}
	if headSmart.Headers["x-ms-access-tier"] != "Smart" {
		t.Fatalf("expected supported Smart tier to be stored, got headers=%v", headSmart.Headers)
	}
}

func TestBlobDataPlaneSetBlobTierCanTargetSnapshot(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", []byte("snapshot content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot id, got headers=%v", snapshotResp.Headers)
	}

	setSnapshotTier, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?comp=tier&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-access-tier": "Cool",
	}))
	if err != nil {
		t.Fatalf("set snapshot blob tier returned error: %v", err)
	}
	if setSnapshotTier.StatusCode != http.StatusOK {
		t.Fatalf("expected snapshot Set Blob Tier status 200, got %d body=%s", setSnapshotTier.StatusCode, string(setSnapshotTier.RawBody))
	}

	headSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tiered.txt?snapshot="+url.QueryEscape(snapshot), nil, nil))
	if err != nil {
		t.Fatalf("head snapshot returned error: %v", err)
	}
	if headSnapshot.Headers["x-ms-access-tier"] != "Cool" || headSnapshot.Headers["x-ms-access-tier-change-time"] == "" {
		t.Fatalf("expected snapshot tier headers, got %v", headSnapshot.Headers)
	}

	headBase, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tiered.txt", nil, nil))
	if err != nil {
		t.Fatalf("head base returned error: %v", err)
	}
	if headBase.Headers["x-ms-access-tier"] != "" || headBase.Headers["x-ms-access-tier-change-time"] != "" {
		t.Fatalf("expected snapshot Set Blob Tier not to mutate base tier headers, got %v", headBase.Headers)
	}
}

func TestBlobDataPlaneSetBlobTierRejectsSnapshotBefore20191212(t *testing.T) {
	svc := storage.New()
	blobURL := "http://localhost:4577/devstoreaccount1/docs/tiered.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL, []byte("snapshot content"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("create snapshot returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot id, got headers=%v", snapshotResp.Headers)
	}

	rejected, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=tier&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-access-tier": "Cool",
		"x-ms-version":     "2019-07-07",
	}))
	if err != nil {
		t.Fatalf("set old-version snapshot tier returned error: %v", err)
	}
	if rejected.StatusCode != http.StatusBadRequest || rejected.Headers["x-ms-error-code"] != "FeatureVersionMismatch" {
		t.Fatalf("expected old-version snapshot tier to return 400 FeatureVersionMismatch, got status=%d headers=%v body=%s", rejected.StatusCode, rejected.Headers, string(rejected.RawBody))
	}

	headSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodHead, blobURL+"?snapshot="+url.QueryEscape(snapshot), nil, nil))
	if err != nil {
		t.Fatalf("head snapshot after rejected tier returned error: %v", err)
	}
	if headSnapshot.Headers["x-ms-access-tier"] != "" || headSnapshot.Headers["x-ms-access-tier-change-time"] != "" {
		t.Fatalf("expected rejected old-version Set Blob Tier not to mutate snapshot tier headers, got %v", headSnapshot.Headers)
	}
}

func TestBlobDataPlanePutBlobStoresAzureContentPropertyHeaders(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))

	putBlob, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/report.json", []byte(`{"ok":true}`), map[string]string{
		"x-ms-blob-type":                "BlockBlob",
		"x-ms-blob-content-type":        "application/json",
		"x-ms-blob-cache-control":       "no-cache",
		"x-ms-blob-content-language":    "en-US",
		"x-ms-blob-content-encoding":    "br",
		"x-ms-blob-content-md5":         "abc123==",
		"x-ms-blob-content-disposition": `inline; filename="report.json"`,
	}))
	if err != nil {
		t.Fatalf("put blob returned error: %v", err)
	}
	if putBlob.StatusCode != http.StatusCreated {
		t.Fatalf("expected put blob status 201, got %d; body=%s", putBlob.StatusCode, string(putBlob.RawBody))
	}

	headBlob, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/report.json", nil, nil))
	if err != nil {
		t.Fatalf("head blob returned error: %v", err)
	}
	if headBlob.Headers["Content-Type"] != "application/json" ||
		headBlob.Headers["Cache-Control"] != "no-cache" ||
		headBlob.Headers["Content-Language"] != "en-US" ||
		headBlob.Headers["Content-Encoding"] != "br" ||
		headBlob.Headers["Content-MD5"] != "abc123==" ||
		headBlob.Headers["Content-Disposition"] != `inline; filename="report.json"` {
		t.Fatalf("expected Put Blob x-ms content properties on HEAD, got %v", headBlob.Headers)
	}
}

func TestBlobDataPlanePutBlockListCommitsStagedBlocks(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))

	firstBlockID := "YmxvY2stMQ=="
	secondBlockID := "YmxvY2stMg=="
	stageFirst, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=block&blockid="+url.QueryEscape(firstBlockID), []byte("hello "), nil))
	if err != nil {
		t.Fatalf("put first block returned error: %v", err)
	}
	if stageFirst.StatusCode != http.StatusCreated {
		t.Fatalf("expected first block status 201, got %d; body=%s", stageFirst.StatusCode, string(stageFirst.RawBody))
	}
	stageSecond, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=block&blockid="+url.QueryEscape(secondBlockID), []byte("blocks"), nil))
	if err != nil {
		t.Fatalf("put second block returned error: %v", err)
	}
	if stageSecond.StatusCode != http.StatusCreated {
		t.Fatalf("expected second block status 201, got %d; body=%s", stageSecond.StatusCode, string(stageSecond.RawBody))
	}

	commitBody := []byte(`<BlockList><Latest>` + firstBlockID + `</Latest><Latest>` + secondBlockID + `</Latest></BlockList>`)
	commit, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=blocklist", commitBody, map[string]string{
		"x-ms-blob-content-type": "text/plain",
		"x-ms-meta-owner":        "blocks",
	}))
	if err != nil {
		t.Fatalf("put block list returned error: %v", err)
	}
	if commit.StatusCode != http.StatusCreated {
		t.Fatalf("expected block list commit status 201, got %d; body=%s", commit.StatusCode, string(commit.RawBody))
	}
	if commit.Headers["ETag"] == "" {
		t.Fatalf("expected committed block list to return ETag, headers=%v", commit.Headers)
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/chunked.txt", nil, nil))
	if err != nil {
		t.Fatalf("get committed blob returned error: %v", err)
	}
	if string(getBlob.RawBody) != "hello blocks" || getBlob.RawContentType != "text/plain" {
		t.Fatalf("expected committed staged block content and content type, contentType=%q body=%q", getBlob.RawContentType, string(getBlob.RawBody))
	}
	if getBlob.Headers["x-ms-meta-owner"] != "blocks" || getBlob.Headers["x-ms-blob-type"] != "BlockBlob" || getBlob.Headers["Content-Length"] != "12" {
		t.Fatalf("expected block blob headers and metadata after commit, got %v", getBlob.Headers)
	}
}

func TestBlobDataPlanePutBlockFromURLStagesSourceRange(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs"
	sourceURL := baseURL + "/source.txt"
	destinationURL := baseURL + "/chunked-from-url.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, []byte("abcdefghijklmnopqrstuvwxyz"), map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-blob-type":         "BlockBlob",
		"x-ms-blob-content-type": "text/plain",
	}))

	blockID := "ZnJvbS11cmw="
	stageFromURL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL+"?comp=block&blockid="+url.QueryEscape(blockID), nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-copy-source":  sourceURL,
		"x-ms-source-range": "bytes=6-10",
	}))
	if err != nil {
		t.Fatalf("put block from url returned error: %v", err)
	}
	if stageFromURL.StatusCode != http.StatusCreated || stageFromURL.Headers["ETag"] == "" || len(stageFromURL.RawBody) != 0 {
		t.Fatalf("expected Put Block From URL status 201 with storage headers and no body, got status=%d headers=%v body=%s", stageFromURL.StatusCode, stageFromURL.Headers, string(stageFromURL.RawBody))
	}

	blockList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, destinationURL+"?comp=blocklist&blocklisttype=uncommitted", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get uncommitted block list returned error: %v", err)
	}
	if blockList.StatusCode != http.StatusOK || !strings.Contains(string(blockList.RawBody), "<Size>5</Size>") {
		t.Fatalf("expected uncommitted block list to show ranged source size 5, got status=%d body=%s", blockList.StatusCode, string(blockList.RawBody))
	}

	commitBody := []byte(`<BlockList><Latest>` + blockID + `</Latest></BlockList>`)
	commit, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL+"?comp=blocklist", commitBody, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-blob-content-type": "text/plain",
	}))
	if err != nil {
		t.Fatalf("put block list after url stage returned error: %v", err)
	}
	if commit.StatusCode != http.StatusCreated {
		t.Fatalf("expected block list commit status 201, got %d body=%s", commit.StatusCode, string(commit.RawBody))
	}
	getDestination, err := svc.HandleRequest(storageCtx(t, http.MethodGet, destinationURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get committed blob returned error: %v", err)
	}
	if getDestination.StatusCode != http.StatusOK || string(getDestination.RawBody) != "ghijk" {
		t.Fatalf("expected committed blob to contain source range, got status=%d body=%q", getDestination.StatusCode, string(getDestination.RawBody))
	}
}

func TestBlobDataPlanePutBlockListCanReuseCommittedBlocks(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs"
	blobURL := baseURL + "/patchable.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=container", nil, nil))

	firstBlockID := "YmxvY2stMQ=="
	secondBlockID := "YmxvY2stMg=="
	thirdBlockID := "YmxvY2stMw=="
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=block&blockid="+url.QueryEscape(firstBlockID), []byte("keep-"), nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=block&blockid="+url.QueryEscape(secondBlockID), []byte("remove"), nil))
	initialCommit := []byte(`<BlockList><Latest>` + firstBlockID + `</Latest><Latest>` + secondBlockID + `</Latest></BlockList>`)
	initial, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=blocklist", initialCommit, map[string]string{
		"x-ms-blob-content-type": "text/plain",
	}))
	if err != nil {
		t.Fatalf("initial put block list returned error: %v", err)
	}
	if initial.StatusCode != http.StatusCreated {
		t.Fatalf("expected initial commit status 201, got %d body=%s", initial.StatusCode, string(initial.RawBody))
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=block&blockid="+url.QueryEscape(thirdBlockID), []byte("new"), nil))
	updateCommit := []byte(`<BlockList><Committed>` + firstBlockID + `</Committed><Uncommitted>` + thirdBlockID + `</Uncommitted></BlockList>`)
	updated, err := svc.HandleRequest(storageCtx(t, http.MethodPut, blobURL+"?comp=blocklist", updateCommit, map[string]string{
		"x-ms-blob-content-type": "text/plain",
	}))
	if err != nil {
		t.Fatalf("updated put block list returned error: %v", err)
	}
	if updated.StatusCode != http.StatusCreated {
		t.Fatalf("expected updated commit status 201, got %d body=%s", updated.StatusCode, string(updated.RawBody))
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, blobURL, nil, nil))
	if err != nil {
		t.Fatalf("get updated blob returned error: %v", err)
	}
	if getBlob.StatusCode != http.StatusOK || string(getBlob.RawBody) != "keep-new" {
		t.Fatalf("expected mixed committed/uncommitted block list to produce keep-new, got status=%d body=%q", getBlob.StatusCode, string(getBlob.RawBody))
	}
	blockList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, blobURL+"?comp=blocklist&blocklisttype=all", nil, nil))
	if err != nil {
		t.Fatalf("get block list returned error: %v", err)
	}
	body := string(blockList.RawBody)
	if strings.Contains(body, secondBlockID) || !strings.Contains(body, firstBlockID) || !strings.Contains(body, thirdBlockID) {
		t.Fatalf("expected committed block list to keep first and third blocks only, got %s", body)
	}
}

func TestBlobDataPlaneGetBlockListReturnsCommittedAndUncommittedBlocks(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))

	firstBlockID := "YmxvY2stMQ=="
	secondBlockID := "YmxvY2stMg=="
	thirdBlockID := "YmxvY2stMw=="
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=block&blockid="+url.QueryEscape(firstBlockID), []byte("hello "), nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=block&blockid="+url.QueryEscape(secondBlockID), []byte("blocks"), nil))
	commitBody := []byte(`<BlockList><Latest>` + firstBlockID + `</Latest><Latest>` + secondBlockID + `</Latest></BlockList>`)
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=blocklist", commitBody, map[string]string{
		"x-ms-blob-content-type": "text/plain",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=block&blockid="+url.QueryEscape(thirdBlockID), []byte("uncommitted"), nil))

	blockList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/chunked.txt?comp=blocklist&blocklisttype=all", nil, nil))
	if err != nil {
		t.Fatalf("get block list returned error: %v", err)
	}
	if blockList.StatusCode != http.StatusOK {
		t.Fatalf("expected get block list status 200, got %d; body=%s", blockList.StatusCode, string(blockList.RawBody))
	}
	if blockList.RawContentType != "application/xml" ||
		blockList.Headers["x-ms-blob-content-length"] != "12" ||
		blockList.Headers["ETag"] == "" ||
		blockList.Headers["Last-Modified"] == "" {
		t.Fatalf("expected Azure block list headers, contentType=%q headers=%v", blockList.RawContentType, blockList.Headers)
	}

	body := string(blockList.RawBody)
	expectedFragments := []string{
		"<CommittedBlocks>",
		"<Name>" + firstBlockID + "</Name>",
		"<Size>6</Size>",
		"<Name>" + secondBlockID + "</Name>",
		"<UncommittedBlocks>",
		"<Name>" + thirdBlockID + "</Name>",
		"<Size>11</Size>",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected block list body to contain %q, got %s", fragment, body)
		}
	}
	if strings.Index(body, "<Name>"+firstBlockID+"</Name>") > strings.Index(body, "<Name>"+secondBlockID+"</Name>") {
		t.Fatalf("expected committed blocks to preserve commit order, got %s", body)
	}
}

func TestBlobDataPlaneCopyBlobCopiesContentPropertiesAndMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs"
	sourceURL := baseURL + "/source.txt"
	copyURL := baseURL + "/copied.txt"
	metadataURL := baseURL + "/metadata-copy.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, []byte("hello copied blob"), map[string]string{
		"x-ms-version":                  "2023-11-03",
		"x-ms-blob-type":                "BlockBlob",
		"x-ms-blob-content-type":        "text/plain",
		"x-ms-blob-cache-control":       "max-age=60",
		"x-ms-blob-content-encoding":    "gzip",
		"x-ms-blob-content-language":    "en-US",
		"x-ms-blob-content-md5":         "abc123==",
		"x-ms-blob-content-disposition": `inline; filename="source.txt"`,
		"x-ms-meta-owner":               "source",
	}))

	copyResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, copyURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("copy blob returned error: %v", err)
	}
	if copyResp.StatusCode != http.StatusAccepted ||
		copyResp.Headers["x-ms-copy-id"] == "" ||
		copyResp.Headers["x-ms-copy-status"] != "success" ||
		copyResp.Headers["ETag"] == "" ||
		copyResp.Headers["Last-Modified"] == "" ||
		len(copyResp.RawBody) != 0 {
		t.Fatalf("expected copy blob to return 202 with copy headers and no body, got status=%d headers=%v body=%s", copyResp.StatusCode, copyResp.Headers, string(copyResp.RawBody))
	}

	getCopy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, copyURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get copied blob returned error: %v", err)
	}
	if getCopy.StatusCode != http.StatusOK || string(getCopy.RawBody) != "hello copied blob" || getCopy.RawContentType != "text/plain" {
		t.Fatalf("expected copied blob content and content type, got status=%d contentType=%q body=%q", getCopy.StatusCode, getCopy.RawContentType, string(getCopy.RawBody))
	}
	expectedHeaders := map[string]string{
		"x-ms-meta-owner":     "source",
		"Content-Length":      "17",
		"Content-Type":        "text/plain",
		"Cache-Control":       "max-age=60",
		"Content-Encoding":    "gzip",
		"Content-Language":    "en-US",
		"Content-MD5":         "abc123==",
		"Content-Disposition": `inline; filename="source.txt"`,
		"x-ms-copy-id":        copyResp.Headers["x-ms-copy-id"],
		"x-ms-copy-status":    "success",
		"x-ms-copy-source":    sourceURL,
	}
	for key, want := range expectedHeaders {
		if getCopy.Headers[key] != want {
			t.Fatalf("expected copied header %s=%q, got headers=%v", key, want, getCopy.Headers)
		}
	}

	metadataCopy, err := svc.HandleRequest(storageCtx(t, http.MethodPut, metadataURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
		"x-ms-meta-owner":  "destination",
	}))
	if err != nil {
		t.Fatalf("metadata replacement copy returned error: %v", err)
	}
	if metadataCopy.StatusCode != http.StatusAccepted {
		t.Fatalf("expected metadata replacement copy status 202, got %d body=%s", metadataCopy.StatusCode, string(metadataCopy.RawBody))
	}
	headMetadataCopy, err := svc.HandleRequest(storageCtx(t, http.MethodHead, metadataURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head metadata replacement copy returned error: %v", err)
	}
	if headMetadataCopy.Headers["x-ms-meta-owner"] != "destination" {
		t.Fatalf("expected destination metadata to replace source metadata, got %v", headMetadataCopy.Headers)
	}
}

func TestBlobDataPlaneAbortCopyRejectsCompletedCopy(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs"
	sourceURL := baseURL + "/source.txt"
	copyURL := baseURL + "/copied.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, []byte("copy source"), map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
	}))

	copyResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, copyURL, nil, map[string]string{
		"x-ms-version":     "2023-11-03",
		"x-ms-copy-source": sourceURL,
	}))
	if err != nil {
		t.Fatalf("copy blob returned error: %v", err)
	}
	copyID := copyResp.Headers["x-ms-copy-id"]
	if copyResp.StatusCode != http.StatusAccepted || copyID == "" || copyResp.Headers["x-ms-copy-status"] != "success" {
		t.Fatalf("expected copy blob success headers, got status=%d headers=%v body=%s", copyResp.StatusCode, copyResp.Headers, string(copyResp.RawBody))
	}

	abortCompleted, err := svc.HandleRequest(storageCtx(t, http.MethodPut, copyURL+"?comp=copy&copyid="+url.QueryEscape(copyID), nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-copy-action":       "abort",
		"x-ms-client-request-id": "abort-completed-copy",
	}))
	if err != nil {
		t.Fatalf("abort completed copy returned error: %v", err)
	}
	if abortCompleted.StatusCode != http.StatusConflict || !strings.Contains(string(abortCompleted.RawBody), "NoPendingCopyOperation") {
		t.Fatalf("expected abort completed copy to return 409 NoPendingCopyOperation, got status=%d headers=%v body=%s", abortCompleted.StatusCode, abortCompleted.Headers, string(abortCompleted.RawBody))
	}
	if abortCompleted.Headers["x-ms-client-request-id"] != "abort-completed-copy" || abortCompleted.Headers["x-ms-request-id"] == "" {
		t.Fatalf("expected abort copy conflict to echo request IDs, got %v", abortCompleted.Headers)
	}

	getCopy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, copyURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get copy after abort conflict returned error: %v", err)
	}
	if getCopy.StatusCode != http.StatusOK || string(getCopy.RawBody) != "copy source" || getCopy.Headers["x-ms-copy-status"] != "success" {
		t.Fatalf("expected failed abort not to mutate completed copy, got status=%d headers=%v body=%s", getCopy.StatusCode, getCopy.Headers, string(getCopy.RawBody))
	}
}

func TestBlobDataPlanePutBlobFromURLCreatesBlockBlobWithoutCopyStatus(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs"
	sourceURL := baseURL + "/source.json"
	destinationURL := baseURL + "/from-url.json"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, sourceURL, []byte(`{"from":"url"}`), map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-blob-type":         "BlockBlob",
		"x-ms-blob-content-type": "text/plain",
		"x-ms-meta-owner":        "source",
	}))

	putFromURL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, destinationURL, nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-copy-source":       sourceURL,
		"x-ms-blob-type":         "BlockBlob",
		"x-ms-blob-content-type": "application/json",
		"x-ms-meta-owner":        "destination",
	}))
	if err != nil {
		t.Fatalf("put blob from url returned error: %v", err)
	}
	if putFromURL.StatusCode != http.StatusCreated ||
		putFromURL.Headers["ETag"] == "" ||
		putFromURL.Headers["Last-Modified"] == "" ||
		putFromURL.Headers["x-ms-copy-id"] != "" ||
		putFromURL.Headers["x-ms-copy-status"] != "" ||
		len(putFromURL.RawBody) != 0 {
		t.Fatalf("expected Put Blob From URL to return 201 with storage headers and no copy status, got status=%d headers=%v body=%s", putFromURL.StatusCode, putFromURL.Headers, string(putFromURL.RawBody))
	}

	getDestination, err := svc.HandleRequest(storageCtx(t, http.MethodGet, destinationURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get destination blob returned error: %v", err)
	}
	if getDestination.StatusCode != http.StatusOK ||
		string(getDestination.RawBody) != `{"from":"url"}` ||
		getDestination.RawContentType != "application/json" ||
		getDestination.Headers["x-ms-meta-owner"] != "destination" ||
		getDestination.Headers["x-ms-copy-id"] != "" ||
		getDestination.Headers["x-ms-copy-status"] != "" {
		t.Fatalf("expected Put Blob From URL to create a normal block blob from source bytes, got status=%d contentType=%q headers=%v body=%q", getDestination.StatusCode, getDestination.RawContentType, getDestination.Headers, string(getDestination.RawBody))
	}
}

func TestBlobDataPlaneSetAndGetBlobTags(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tagged.txt", []byte("tagged blob"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))
	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tagged.txt", nil, nil))
	if err != nil {
		t.Fatalf("head blob before tags returned error: %v", err)
	}

	tagBody := []byte(`<Tags><TagSet><Tag><Key>env</Key><Value>test</Value></Tag><Tag><Key>release</Key><Value>2026</Value></Tag></TagSet></Tags>`)
	setTags, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/tagged.txt?comp=tags", tagBody, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
	}))
	if err != nil {
		t.Fatalf("set blob tags returned error: %v", err)
	}
	if setTags.StatusCode != http.StatusNoContent || len(setTags.RawBody) != 0 {
		t.Fatalf("expected set tags status 204 without body, got %d body=%s", setTags.StatusCode, string(setTags.RawBody))
	}

	headAfter, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/tagged.txt", nil, nil))
	if err != nil {
		t.Fatalf("head blob after tags returned error: %v", err)
	}
	if headAfter.Headers["ETag"] != headBefore.Headers["ETag"] || headAfter.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected Set Blob Tags to preserve blob ETag and Last-Modified, before=%v after=%v", headBefore.Headers, headAfter.Headers)
	}
	if headAfter.Headers["x-ms-tag-count"] != "2" {
		t.Fatalf("expected Get Blob Properties to report tag count 2, got headers=%v", headAfter.Headers)
	}

	getBlob, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/tagged.txt", nil, nil))
	if err != nil {
		t.Fatalf("get tagged blob returned error: %v", err)
	}
	if getBlob.Headers["x-ms-tag-count"] != "2" {
		t.Fatalf("expected Get Blob to report tag count 2, got headers=%v", getBlob.Headers)
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/tagged.txt?comp=metadata", nil, nil))
	if err != nil {
		t.Fatalf("get tagged blob metadata returned error: %v", err)
	}
	if _, ok := getMetadata.Headers["x-ms-tag-count"]; ok {
		t.Fatalf("expected Get Blob Metadata not to report tag count, got headers=%v", getMetadata.Headers)
	}

	getTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/tagged.txt?comp=tags", nil, nil))
	if err != nil {
		t.Fatalf("get blob tags returned error: %v", err)
	}
	if getTags.StatusCode != http.StatusOK || getTags.RawContentType != "application/xml" {
		t.Fatalf("expected get tags status 200 application/xml, got %d contentType=%q body=%s", getTags.StatusCode, getTags.RawContentType, string(getTags.RawBody))
	}
	body := string(getTags.RawBody)
	expectedFragments := []string{
		"<Tags>",
		"<TagSet>",
		"<Key>env</Key>",
		"<Value>test</Value>",
		"<Key>release</Key>",
		"<Value>2026</Value>",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected tags response to contain %q, got %s", fragment, body)
		}
	}
}

func TestBlobDataPlaneGetSnapshotBlobTagsReturnsPointInTimeTags(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs/tag-snapshot.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, []byte("tagged snapshot"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	initialTags := []byte(`<Tags><TagSet><Tag><Key>phase</Key><Value>initial</Value></Tag><Tag><Key>owner</Key><Value>snapshot</Value></Tag></TagSet></Tags>`)
	setInitialTags, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags", initialTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set initial tags returned error: %v", err)
	}
	if setInitialTags.StatusCode != http.StatusNoContent {
		t.Fatalf("expected initial set tags status 204, got %d body=%s", setInitialTags.StatusCode, string(setInitialTags.RawBody))
	}

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot blob returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot response to include x-ms-snapshot, got headers=%v", snapshotResp.Headers)
	}

	updatedTags := []byte(`<Tags><TagSet><Tag><Key>phase</Key><Value>base-updated</Value></Tag><Tag><Key>current</Key><Value>true</Value></Tag></TagSet></Tags>`)
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags", updatedTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2023-11-03",
	}))

	getSnapshotTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=tags&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot tags returned error: %v", err)
	}
	body := string(getSnapshotTags.RawBody)
	if getSnapshotTags.StatusCode != http.StatusOK ||
		!strings.Contains(body, "<Key>phase</Key><Value>initial</Value>") ||
		!strings.Contains(body, "<Key>owner</Key><Value>snapshot</Value>") ||
		strings.Contains(body, "base-updated") ||
		strings.Contains(body, "<Key>current</Key>") {
		t.Fatalf("expected snapshot tag read to return point-in-time tags, got status=%d headers=%v body=%s", getSnapshotTags.StatusCode, getSnapshotTags.Headers, body)
	}
}

func TestBlobDataPlaneSetSnapshotBlobTagsDoesNotMutateBaseBlob(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs/tag-snapshot-write.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, []byte("snapshot tag write"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	baseTags := []byte(`<Tags><TagSet><Tag><Key>phase</Key><Value>base</Value></Tag></TagSet></Tags>`)
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags", baseTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2023-11-03",
	}))
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot blob returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot response to include x-ms-snapshot, got headers=%v", snapshotResp.Headers)
	}

	snapshotTags := []byte(`<Tags><TagSet><Tag><Key>phase</Key><Value>snapshot-updated</Value></Tag><Tag><Key>legal</Key><Value>hold</Value></Tag></TagSet></Tags>`)
	setSnapshotTags, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags&snapshot="+url.QueryEscape(snapshot), snapshotTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set snapshot tags returned error: %v", err)
	}
	if setSnapshotTags.StatusCode != http.StatusNoContent || len(setSnapshotTags.RawBody) != 0 {
		t.Fatalf("expected set snapshot tags status 204 without body, got %d body=%s", setSnapshotTags.StatusCode, string(setSnapshotTags.RawBody))
	}

	getSnapshotTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=tags&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot tags returned error: %v", err)
	}
	snapshotBody := string(getSnapshotTags.RawBody)
	if !strings.Contains(snapshotBody, "<Key>phase</Key><Value>snapshot-updated</Value>") ||
		!strings.Contains(snapshotBody, "<Key>legal</Key><Value>hold</Value>") {
		t.Fatalf("expected snapshot tags to be updated, got status=%d headers=%v body=%s", getSnapshotTags.StatusCode, getSnapshotTags.Headers, snapshotBody)
	}

	getBaseTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=tags", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get base tags returned error: %v", err)
	}
	baseBody := string(getBaseTags.RawBody)
	if !strings.Contains(baseBody, "<Key>phase</Key><Value>base</Value>") ||
		strings.Contains(baseBody, "snapshot-updated") ||
		strings.Contains(baseBody, "<Key>legal</Key>") {
		t.Fatalf("expected snapshot tag write not to mutate base tags, got status=%d headers=%v body=%s", getBaseTags.StatusCode, getBaseTags.Headers, baseBody)
	}
}

func TestBlobDataPlaneSetSnapshotBlobTagsRejectsOlderServiceVersion(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs/tag-snapshot-version.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, []byte("snapshot tag version"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=snapshot", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("snapshot blob returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshot == "" {
		t.Fatalf("expected snapshot response to include x-ms-snapshot, got headers=%v", snapshotResp.Headers)
	}

	rejectedTags := []byte(`<Tags><TagSet><Tag><Key>phase</Key><Value>too-old</Value></Tag></TagSet></Tags>`)
	rejectedSet, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags&snapshot="+url.QueryEscape(snapshot), rejectedTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2019-12-12",
	}))
	if err != nil {
		t.Fatalf("old-version set snapshot tags returned error: %v", err)
	}
	if rejectedSet.StatusCode != http.StatusBadRequest || rejectedSet.Headers["x-ms-error-code"] != "FeatureVersionMismatch" {
		t.Fatalf("expected old-version snapshot tag update to fail FeatureVersionMismatch, got status=%d headers=%v body=%s", rejectedSet.StatusCode, rejectedSet.Headers, string(rejectedSet.RawBody))
	}

	getSnapshotTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=tags&snapshot="+url.QueryEscape(snapshot), nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get snapshot tags returned error: %v", err)
	}
	if strings.Contains(string(getSnapshotTags.RawBody), "too-old") {
		t.Fatalf("expected rejected old-version update not to mutate snapshot tags, got body=%s", string(getSnapshotTags.RawBody))
	}
}

func TestBlobDataPlaneSetBlobTagsRejectsInvalidTagSets(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1/docs/invalid-tags.txt"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, []byte("invalid tag preservation"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	validTags := []byte(`<Tags><TagSet><Tag><Key>valid</Key><Value>kept</Value></Tag></TagSet></Tags>`)
	setValidTags, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags", validTags, map[string]string{
		"Content-Type": "application/xml; charset=UTF-8",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid tags returned error: %v", err)
	}
	if setValidTags.StatusCode != http.StatusNoContent {
		t.Fatalf("expected valid tags status 204, got %d body=%s", setValidTags.StatusCode, string(setValidTags.RawBody))
	}

	longKey := strings.Repeat("k", 129)
	longValue := strings.Repeat("v", 257)
	elevenTags := `<Tags><TagSet>` +
		`<Tag><Key>k01</Key><Value>v</Value></Tag><Tag><Key>k02</Key><Value>v</Value></Tag>` +
		`<Tag><Key>k03</Key><Value>v</Value></Tag><Tag><Key>k04</Key><Value>v</Value></Tag>` +
		`<Tag><Key>k05</Key><Value>v</Value></Tag><Tag><Key>k06</Key><Value>v</Value></Tag>` +
		`<Tag><Key>k07</Key><Value>v</Value></Tag><Tag><Key>k08</Key><Value>v</Value></Tag>` +
		`<Tag><Key>k09</Key><Value>v</Value></Tag><Tag><Key>k10</Key><Value>v</Value></Tag>` +
		`<Tag><Key>k11</Key><Value>v</Value></Tag></TagSet></Tags>`
	invalidBodies := map[string]string{
		"too many tags": elevenTags,
		"empty key":     `<Tags><TagSet><Tag><Key></Key><Value>v</Value></Tag></TagSet></Tags>`,
		"long key":      `<Tags><TagSet><Tag><Key>` + longKey + `</Key><Value>v</Value></Tag></TagSet></Tags>`,
		"long value":    `<Tags><TagSet><Tag><Key>k</Key><Value>` + longValue + `</Value></Tag></TagSet></Tags>`,
		"invalid key":   `<Tags><TagSet><Tag><Key>bad,key</Key><Value>v</Value></Tag></TagSet></Tags>`,
		"invalid value": `<Tags><TagSet><Tag><Key>k</Key><Value>bad,value</Value></Tag></TagSet></Tags>`,
	}
	for name, body := range invalidBodies {
		t.Run(name, func(t *testing.T) {
			rejected, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=tags", []byte(body), map[string]string{
				"Content-Type": "application/xml; charset=UTF-8",
				"x-ms-version": "2023-11-03",
			}))
			if err != nil {
				t.Fatalf("set invalid tags returned error: %v", err)
			}
			if rejected.StatusCode != http.StatusBadRequest || rejected.Headers["x-ms-error-code"] == "" {
				t.Fatalf("expected invalid tag set to fail 400 with storage error header, got status=%d headers=%v body=%s", rejected.StatusCode, rejected.Headers, string(rejected.RawBody))
			}
		})
	}

	getTags, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=tags", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get tags after invalid updates returned error: %v", err)
	}
	body := string(getTags.RawBody)
	if !strings.Contains(body, "<Key>valid</Key><Value>kept</Value>") ||
		strings.Contains(body, "bad,value") ||
		strings.Contains(body, "<Key>k11</Key>") {
		t.Fatalf("expected invalid tag sets not to mutate stored tags, got status=%d headers=%v body=%s", getTags.StatusCode, getTags.Headers, body)
	}
}

func TestBlobDataPlaneFindBlobsByTags(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs?restype=container", nil, nil))

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/match.txt", []byte("match"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/skip.txt", []byte("skip"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/match.txt", []byte("wrong container"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))

	tagBody := []byte(`<Tags><TagSet><Tag><Key>env</Key><Value>test</Value></Tag><Tag><Key>release</Key><Value>2026</Value></Tag></TagSet></Tags>`)
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/match.txt?comp=tags", tagBody, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/skip.txt?comp=tags", []byte(`<Tags><TagSet><Tag><Key>env</Key><Value>prod</Value></Tag></TagSet></Tags>`), nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/logs/match.txt?comp=tags", tagBody, nil))

	where := url.QueryEscape(`@container='docs' AND "env"='test'`)
	found, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1?comp=blobs&where="+where, nil, nil))
	if err != nil {
		t.Fatalf("find blobs by tags returned error: %v", err)
	}
	if found.StatusCode != http.StatusOK || found.RawContentType != "application/xml" {
		t.Fatalf("expected find blobs status 200 application/xml, got %d contentType=%q body=%s", found.StatusCode, found.RawContentType, string(found.RawBody))
	}
	if found.Headers["Content-Length"] == "" || found.Headers["x-ms-version"] == "" {
		t.Fatalf("expected find blobs response headers, got %v", found.Headers)
	}

	body := string(found.RawBody)
	expectedFragments := []string{
		`<EnumerationResults ServiceEndpoint="https://devstoreaccount1.blob.core.windows.net/">`,
		`<Where>@container=&#39;docs&#39; AND &#34;env&#34;=&#39;test&#39;</Where>`,
		"<Name>match.txt</Name>",
		"<ContainerName>docs</ContainerName>",
		"<Key>env</Key>",
		"<Value>test</Value>",
		"<Key>release</Key>",
		"<Value>2026</Value>",
	}
	for _, fragment := range expectedFragments {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected find response to contain %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "<Name>skip.txt</Name>") || strings.Contains(body, "<ContainerName>logs</ContainerName>") {
		t.Fatalf("expected find response to filter non-matching blobs, got %s", body)
	}
}

func TestBlobDataPlaneLeaseAcquireReleaseAndWriteEnforcement(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt", []byte("leased blob"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	headBefore, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/leased.txt", nil, nil))
	if err != nil {
		t.Fatalf("head blob before lease returned error: %v", err)
	}
	leaseID := "11111111-1111-1111-1111-111111111111"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))
	if err != nil {
		t.Fatalf("acquire lease returned error: %v", err)
	}
	if acquire.StatusCode != http.StatusCreated ||
		acquire.Headers["x-ms-lease-id"] != leaseID ||
		acquire.Headers["ETag"] != headBefore.Headers["ETag"] ||
		acquire.Headers["Last-Modified"] != headBefore.Headers["Last-Modified"] {
		t.Fatalf("expected acquire lease headers without blob property mutation, status=%d headers=%v before=%v", acquire.StatusCode, acquire.Headers, headBefore.Headers)
	}

	headLeased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/leased.txt", nil, nil))
	if err != nil {
		t.Fatalf("head leased blob returned error: %v", err)
	}
	if headLeased.Headers["x-ms-lease-status"] != "locked" || headLeased.Headers["x-ms-lease-state"] != "leased" {
		t.Fatalf("expected locked leased headers after acquire, got %v", headLeased.Headers)
	}

	putWithoutLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt", []byte("blocked"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
	}))
	if err != nil {
		t.Fatalf("put without lease returned error: %v", err)
	}
	if putWithoutLease.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected write without lease to fail with 412, got %d body=%s", putWithoutLease.StatusCode, string(putWithoutLease.RawBody))
	}

	putWithLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt", []byte("allowed"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"x-ms-lease-id":  leaseID,
		"Content-Type":   "text/plain",
	}))
	if err != nil {
		t.Fatalf("put with lease returned error: %v", err)
	}
	if putWithLease.StatusCode != http.StatusCreated {
		t.Fatalf("expected write with lease to succeed, got %d body=%s", putWithLease.StatusCode, string(putWithLease.RawBody))
	}

	release, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action": "release",
		"x-ms-lease-id":     leaseID,
	}))
	if err != nil {
		t.Fatalf("release lease returned error: %v", err)
	}
	if release.StatusCode != http.StatusOK || release.Headers["ETag"] != putWithLease.Headers["ETag"] {
		t.Fatalf("expected release lease status 200 and current ETag, got %d headers=%v put=%v", release.StatusCode, release.Headers, putWithLease.Headers)
	}

	headReleased, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/leased.txt", nil, nil))
	if err != nil {
		t.Fatalf("head released blob returned error: %v", err)
	}
	if headReleased.Headers["x-ms-lease-status"] != "unlocked" || headReleased.Headers["x-ms-lease-state"] != "available" {
		t.Fatalf("expected unlocked available headers after release, got %v", headReleased.Headers)
	}
}

func TestBlobDataPlaneReadLeaseIDValidation(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt", []byte("leased blob"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "storage",
	}))

	leaseID := "11111111-1111-1111-1111-111111111111"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/leased.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseID,
	}))

	readCases := []struct {
		name   string
		method string
		url    string
	}{
		{name: "get blob", method: http.MethodGet, url: "http://localhost:4577/devstoreaccount1/docs/leased.txt"},
		{name: "get blob properties", method: http.MethodHead, url: "http://localhost:4577/devstoreaccount1/docs/leased.txt"},
		{name: "get blob metadata", method: http.MethodGet, url: "http://localhost:4577/devstoreaccount1/docs/leased.txt?comp=metadata"},
	}
	for _, tc := range readCases {
		t.Run(tc.name, func(t *testing.T) {
			wrongLease, err := svc.HandleRequest(storageCtx(t, tc.method, tc.url, nil, map[string]string{
				"x-ms-lease-id": "22222222-2222-2222-2222-222222222222",
			}))
			if err != nil {
				t.Fatalf("read with mismatched lease returned error: %v", err)
			}
			if wrongLease.StatusCode != http.StatusPreconditionFailed {
				t.Fatalf("expected mismatched read lease to fail 412, got %d body=%s", wrongLease.StatusCode, string(wrongLease.RawBody))
			}

			matchingLease, err := svc.HandleRequest(storageCtx(t, tc.method, tc.url, nil, map[string]string{
				"x-ms-lease-id": leaseID,
			}))
			if err != nil {
				t.Fatalf("read with matching lease returned error: %v", err)
			}
			if matchingLease.StatusCode != http.StatusOK {
				t.Fatalf("expected matching read lease to succeed, got %d body=%s", matchingLease.StatusCode, string(matchingLease.RawBody))
			}
		})
	}
}

func TestBlobDataPlaneReadLeaseIDFailsWhenBlobIsNotLeased(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/open.txt", []byte("open blob"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/open.txt", nil, map[string]string{
		"x-ms-lease-id": "11111111-1111-1111-1111-111111111111",
	}))
	if err != nil {
		t.Fatalf("read available blob with lease header returned error: %v", err)
	}
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("expected read lease ID on available blob to fail 412, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
}

func TestBlobDataPlaneLeaseRenewChangeAndBreak(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt", []byte("leased blob"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	leaseA := "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa"
	leaseB := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	leaseC := "cccccccc-cccc-cccc-cccc-cccccccccccc"
	acquire, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseA,
	}))
	if err != nil {
		t.Fatalf("acquire lease returned error: %v", err)
	}

	renew, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action": "renew",
		"x-ms-lease-id":     leaseA,
	}))
	if err != nil {
		t.Fatalf("renew lease returned error: %v", err)
	}
	if renew.StatusCode != http.StatusOK || renew.Headers["x-ms-lease-id"] != leaseA || renew.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected renew status 200 with current lease and unchanged ETag, got status=%d headers=%v acquire=%v", renew.StatusCode, renew.Headers, acquire.Headers)
	}

	change, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "change",
		"x-ms-lease-id":          leaseA,
		"x-ms-proposed-lease-id": leaseB,
	}))
	if err != nil {
		t.Fatalf("change lease returned error: %v", err)
	}
	if change.StatusCode != http.StatusOK || change.Headers["x-ms-lease-id"] != leaseB || change.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected change status 200 with proposed lease and unchanged ETag, got status=%d headers=%v acquire=%v", change.StatusCode, change.Headers, acquire.Headers)
	}

	putWithOldLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt", []byte("old lease"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"x-ms-lease-id":  leaseA,
	}))
	if err != nil {
		t.Fatalf("put with old lease returned error: %v", err)
	}
	if putWithOldLease.StatusCode != http.StatusConflict {
		t.Fatalf("expected old lease write to fail with 409, got %d body=%s", putWithOldLease.StatusCode, string(putWithOldLease.RawBody))
	}

	breakLease, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":       "break",
		"x-ms-lease-break-period": "0",
	}))
	if err != nil {
		t.Fatalf("break lease returned error: %v", err)
	}
	if breakLease.StatusCode != http.StatusAccepted || breakLease.Headers["x-ms-lease-time"] != "0" || breakLease.Headers["ETag"] != acquire.Headers["ETag"] {
		t.Fatalf("expected break status 202 with lease time 0 and unchanged ETag, got status=%d headers=%v acquire=%v", breakLease.StatusCode, breakLease.Headers, acquire.Headers)
	}

	headBroken, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/state.txt", nil, nil))
	if err != nil {
		t.Fatalf("head broken lease returned error: %v", err)
	}
	if headBroken.Headers["x-ms-lease-status"] != "unlocked" || headBroken.Headers["x-ms-lease-state"] != "broken" {
		t.Fatalf("expected broken unlocked lease headers after break, got %v", headBroken.Headers)
	}

	acquireAfterBreak, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/state.txt?comp=lease", nil, map[string]string{
		"x-ms-lease-action":      "acquire",
		"x-ms-lease-duration":    "-1",
		"x-ms-proposed-lease-id": leaseC,
	}))
	if err != nil {
		t.Fatalf("acquire after break returned error: %v", err)
	}
	if acquireAfterBreak.StatusCode != http.StatusCreated || acquireAfterBreak.Headers["x-ms-lease-id"] != leaseC {
		t.Fatalf("expected acquire after break to succeed with new lease, got status=%d headers=%v", acquireAfterBreak.StatusCode, acquireAfterBreak.Headers)
	}
}

func TestBlobDataPlaneSnapshotBlobPreservesPointInTimeContent(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/snap.txt", []byte("original"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "text/plain",
		"x-ms-meta-owner": "docs",
	}))

	snapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/snap.txt?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("snapshot blob returned error: %v", err)
	}
	snapshot := snapshotResp.Headers["x-ms-snapshot"]
	if snapshotResp.StatusCode != http.StatusCreated || snapshot == "" || snapshotResp.Headers["ETag"] == "" || snapshotResp.Headers["Last-Modified"] == "" {
		t.Fatalf("expected snapshot status 201 with snapshot, ETag, and last-modified headers, got status=%d headers=%v", snapshotResp.StatusCode, snapshotResp.Headers)
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/snap.txt", []byte("updated"), map[string]string{
		"x-ms-blob-type":  "BlockBlob",
		"Content-Type":    "application/json",
		"x-ms-meta-owner": "updated",
	}))

	getSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/snap.txt?snapshot="+url.QueryEscape(snapshot), nil, nil))
	if err != nil {
		t.Fatalf("get snapshot returned error: %v", err)
	}
	if getSnapshot.StatusCode != http.StatusOK || string(getSnapshot.RawBody) != "original" || getSnapshot.RawContentType != "text/plain" {
		t.Fatalf("expected snapshot GET to return original text blob, got status=%d contentType=%q body=%q", getSnapshot.StatusCode, getSnapshot.RawContentType, string(getSnapshot.RawBody))
	}
	if getSnapshot.Headers["x-ms-snapshot"] != snapshot || getSnapshot.Headers["x-ms-meta-owner"] != "docs" {
		t.Fatalf("expected snapshot headers to include snapshot id and original metadata, got %v", getSnapshot.Headers)
	}

	getBase, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/snap.txt", nil, nil))
	if err != nil {
		t.Fatalf("get base blob returned error: %v", err)
	}
	if getBase.StatusCode != http.StatusOK || string(getBase.RawBody) != "updated" || getBase.RawContentType != "application/json" {
		t.Fatalf("expected base GET to return updated JSON blob, got status=%d contentType=%q body=%q", getBase.StatusCode, getBase.RawContentType, string(getBase.RawBody))
	}

	headSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodHead, "http://localhost:4577/devstoreaccount1/docs/snap.txt?snapshot="+url.QueryEscape(snapshot), nil, nil))
	if err != nil {
		t.Fatalf("head snapshot returned error: %v", err)
	}
	if headSnapshot.StatusCode != http.StatusOK ||
		headSnapshot.Headers["x-ms-snapshot"] != snapshot ||
		headSnapshot.Headers["Content-Length"] != "8" ||
		headSnapshot.Headers["Content-Type"] != "text/plain" ||
		headSnapshot.Headers["x-ms-meta-owner"] != "docs" {
		t.Fatalf("expected snapshot HEAD to return original properties, got status=%d headers=%v", headSnapshot.StatusCode, headSnapshot.Headers)
	}
}

func TestBlobDataPlaneDeleteBlobHonorsSnapshotOptions(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs?restype=container", nil, nil))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/delete.txt", []byte("original"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	firstSnapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/delete.txt?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("create first snapshot returned error: %v", err)
	}
	firstSnapshot := firstSnapshotResp.Headers["x-ms-snapshot"]
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/delete.txt", []byte("updated"), map[string]string{
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	deleteWithoutHeader, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, nil))
	if err != nil {
		t.Fatalf("delete base without snapshot header returned error: %v", err)
	}
	if deleteWithoutHeader.StatusCode != http.StatusConflict || deleteWithoutHeader.Headers["x-ms-error-code"] != "SnapshotsPresent" {
		t.Fatalf("expected base delete with snapshots to fail with 409 SnapshotsPresent, got status=%d headers=%v body=%s", deleteWithoutHeader.StatusCode, deleteWithoutHeader.Headers, string(deleteWithoutHeader.RawBody))
	}

	baseAfterBlockedDelete, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, nil))
	if err != nil {
		t.Fatalf("get base after blocked delete returned error: %v", err)
	}
	if baseAfterBlockedDelete.StatusCode != http.StatusOK || string(baseAfterBlockedDelete.RawBody) != "updated" {
		t.Fatalf("expected blocked delete to preserve base blob, got status=%d body=%q", baseAfterBlockedDelete.StatusCode, string(baseAfterBlockedDelete.RawBody))
	}

	deleteSnapshotsOnly, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, map[string]string{
		"x-ms-delete-snapshots": "only",
	}))
	if err != nil {
		t.Fatalf("delete snapshots only returned error: %v", err)
	}
	if deleteSnapshotsOnly.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete snapshots only status 202, got %d body=%s", deleteSnapshotsOnly.StatusCode, string(deleteSnapshotsOnly.RawBody))
	}

	deletedSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt?snapshot="+url.QueryEscape(firstSnapshot), nil, nil))
	if err != nil {
		t.Fatalf("get deleted snapshot returned error: %v", err)
	}
	if deletedSnapshot.StatusCode != http.StatusNotFound {
		t.Fatalf("expected snapshots-only delete to remove snapshot, got status=%d body=%s", deletedSnapshot.StatusCode, string(deletedSnapshot.RawBody))
	}
	baseAfterOnly, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, nil))
	if err != nil {
		t.Fatalf("get base after snapshots-only delete returned error: %v", err)
	}
	if baseAfterOnly.StatusCode != http.StatusOK || string(baseAfterOnly.RawBody) != "updated" {
		t.Fatalf("expected snapshots-only delete to preserve base blob, got status=%d body=%q", baseAfterOnly.StatusCode, string(baseAfterOnly.RawBody))
	}

	secondSnapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/delete.txt?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("create second snapshot returned error: %v", err)
	}
	secondSnapshot := secondSnapshotResp.Headers["x-ms-snapshot"]
	deleteSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "http://localhost:4577/devstoreaccount1/docs/delete.txt?snapshot="+url.QueryEscape(secondSnapshot), nil, nil))
	if err != nil {
		t.Fatalf("delete individual snapshot returned error: %v", err)
	}
	if deleteSnapshot.StatusCode != http.StatusAccepted {
		t.Fatalf("expected individual snapshot delete status 202, got %d body=%s", deleteSnapshot.StatusCode, string(deleteSnapshot.RawBody))
	}
	baseAfterSnapshotDelete, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, nil))
	if err != nil {
		t.Fatalf("get base after individual snapshot delete returned error: %v", err)
	}
	if baseAfterSnapshotDelete.StatusCode != http.StatusOK || string(baseAfterSnapshotDelete.RawBody) != "updated" {
		t.Fatalf("expected individual snapshot delete to preserve base, got status=%d body=%q", baseAfterSnapshotDelete.StatusCode, string(baseAfterSnapshotDelete.RawBody))
	}

	thirdSnapshotResp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1/docs/delete.txt?comp=snapshot", nil, nil))
	if err != nil {
		t.Fatalf("create third snapshot returned error: %v", err)
	}
	thirdSnapshot := thirdSnapshotResp.Headers["x-ms-snapshot"]
	deleteInclude, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, map[string]string{
		"x-ms-delete-snapshots": "include",
	}))
	if err != nil {
		t.Fatalf("delete base including snapshots returned error: %v", err)
	}
	if deleteInclude.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete include status 202, got %d body=%s", deleteInclude.StatusCode, string(deleteInclude.RawBody))
	}

	missingBase, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt", nil, nil))
	if err != nil {
		t.Fatalf("get base after include delete returned error: %v", err)
	}
	if missingBase.StatusCode != http.StatusNotFound {
		t.Fatalf("expected include delete to remove base, got status=%d body=%s", missingBase.StatusCode, string(missingBase.RawBody))
	}
	missingSnapshot, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "http://localhost:4577/devstoreaccount1/docs/delete.txt?snapshot="+url.QueryEscape(thirdSnapshot), nil, nil))
	if err != nil {
		t.Fatalf("get snapshot after include delete returned error: %v", err)
	}
	if missingSnapshot.StatusCode != http.StatusNotFound {
		t.Fatalf("expected include delete to remove snapshots, got status=%d body=%s", missingSnapshot.StatusCode, string(missingSnapshot.RawBody))
	}
}

func TestBlobGetSupportsRangeAndXMSRangePrecedence(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("hello azure blob"), map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
		"Content-Type":   "text/plain",
	}))

	getRange, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
		"Range":        "bytes=6-10",
		"x-ms-range":   "bytes=0-4",
	}))
	if err != nil {
		t.Fatalf("get ranged blob returned error: %v", err)
	}
	if getRange.StatusCode != http.StatusPartialContent {
		t.Fatalf("expected ranged blob status 206, got %d; body=%s", getRange.StatusCode, string(getRange.RawBody))
	}
	if string(getRange.RawBody) != "hello" {
		t.Fatalf("expected x-ms-range to take precedence over Range, got %q", string(getRange.RawBody))
	}
	if getRange.Headers["Content-Range"] != "bytes 0-4/16" {
		t.Fatalf("unexpected Content-Range: %q", getRange.Headers["Content-Range"])
	}
	if getRange.Headers["Content-Length"] != "5" {
		t.Fatalf("unexpected Content-Length: %q", getRange.Headers["Content-Length"])
	}
	if getRange.Headers["Accept-Ranges"] != "bytes" {
		t.Fatalf("expected Accept-Ranges bytes, got %q", getRange.Headers["Accept-Ranges"])
	}
}

func TestBlobGetSupportsRangeHashHeaders(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/readme.txt", []byte("hello azure blob"), map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-blob-type":         "BlockBlob",
		"x-ms-blob-content-type": "text/plain",
	}))

	rangeBody := []byte("hello")
	rangeMD5 := md5.Sum(rangeBody)
	getMD5, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version":               "2023-11-03",
		"x-ms-range":                 "bytes=0-4",
		"x-ms-range-get-content-md5": "true",
	}))
	if err != nil {
		t.Fatalf("get ranged blob with MD5 returned error: %v", err)
	}
	if getMD5.StatusCode != http.StatusPartialContent || string(getMD5.RawBody) != "hello" {
		t.Fatalf("expected partial ranged blob with MD5, got status=%d body=%q", getMD5.StatusCode, string(getMD5.RawBody))
	}
	if getMD5.Headers["Content-MD5"] != base64.StdEncoding.EncodeToString(rangeMD5[:]) {
		t.Fatalf("expected Content-MD5 for ranged blob, got headers=%v", getMD5.Headers)
	}

	rangeCRC := crc64.Checksum(rangeBody, crc64.MakeTable(crc64.ECMA))
	crcBytes := []byte{
		byte(rangeCRC >> 56),
		byte(rangeCRC >> 48),
		byte(rangeCRC >> 40),
		byte(rangeCRC >> 32),
		byte(rangeCRC >> 24),
		byte(rangeCRC >> 16),
		byte(rangeCRC >> 8),
		byte(rangeCRC),
	}
	getCRC, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/readme.txt", nil, map[string]string{
		"x-ms-version":                 "2023-11-03",
		"Range":                        "bytes=0-4",
		"x-ms-range-get-content-crc64": "true",
	}))
	if err != nil {
		t.Fatalf("get ranged blob with CRC64 returned error: %v", err)
	}
	if getCRC.StatusCode != http.StatusPartialContent || string(getCRC.RawBody) != "hello" {
		t.Fatalf("expected partial ranged blob with CRC64, got status=%d body=%q", getCRC.StatusCode, string(getCRC.RawBody))
	}
	if getCRC.Headers["x-ms-content-crc64"] != base64.StdEncoding.EncodeToString(crcBytes) {
		t.Fatalf("expected x-ms-content-crc64 for ranged blob, got headers=%v", getCRC.Headers)
	}
}

func TestBlobGetValidatesRangeHashHeaders(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	largeBody := []byte(strings.Repeat("x", 4*1024*1024+1))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/large.bin", largeBody, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-blob-type": "BlockBlob",
	}))

	cases := []struct {
		name    string
		headers map[string]string
	}{
		{
			name: "md5 without range",
			headers: map[string]string{
				"x-ms-version":               "2023-11-03",
				"x-ms-range-get-content-md5": "true",
			},
		},
		{
			name: "crc64 without range",
			headers: map[string]string{
				"x-ms-version":                 "2023-11-03",
				"x-ms-range-get-content-crc64": "true",
			},
		},
		{
			name: "md5 and crc64 together",
			headers: map[string]string{
				"x-ms-version":                 "2023-11-03",
				"x-ms-range":                   "bytes=0-4",
				"x-ms-range-get-content-md5":   "true",
				"x-ms-range-get-content-crc64": "true",
			},
		},
		{
			name: "md5 range too large",
			headers: map[string]string{
				"x-ms-version":               "2023-11-03",
				"x-ms-range":                 "bytes=0-4194304",
				"x-ms-range-get-content-md5": "true",
			},
		},
		{
			name: "crc64 range too large",
			headers: map[string]string{
				"x-ms-version":                 "2023-11-03",
				"x-ms-range":                   "bytes=0-4194304",
				"x-ms-range-get-content-crc64": "true",
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs/large.bin", nil, tc.headers))
			if err != nil {
				t.Fatalf("get blob with invalid range hash headers returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid range hash request to fail 400, got %d body length=%d headers=%v", resp.StatusCode, len(resp.RawBody), resp.Headers)
			}
		})
	}
}

func TestBlobListUsesContinuationMarkers(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	for _, name := range []string{"a.txt", "b.txt"} {
		_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/docs/"+name, []byte(name), map[string]string{
			"x-ms-version":   "2023-11-03",
			"x-ms-blob-type": "BlockBlob",
		}))
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs?restype=container&comp=list&maxresults=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("first page returned error: %v", err)
	}
	firstBody := string(firstPage.RawBody)
	if !strings.Contains(firstBody, "<Name>a.txt</Name>") || strings.Contains(firstBody, "<Name>b.txt</Name>") {
		t.Fatalf("unexpected first page body: %s", firstBody)
	}
	if !strings.Contains(firstBody, "<NextMarker>b.txt</NextMarker>") {
		t.Fatalf("expected first page NextMarker, got: %s", firstBody)
	}

	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/docs?restype=container&comp=list&maxresults=1&marker=b.txt", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("second page returned error: %v", err)
	}
	secondBody := string(secondPage.RawBody)
	if strings.Contains(secondBody, "<Name>a.txt</Name>") || !strings.Contains(secondBody, "<Name>b.txt</Name>") {
		t.Fatalf("unexpected second page body: %s", secondBody)
	}
	if strings.Contains(secondBody, "<NextMarker>") {
		t.Fatalf("expected final page without NextMarker, got: %s", secondBody)
	}
}

func TestBlobListSupportsPrefixAndDelimiter(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/assets?restype=container", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	for _, name := range []string{"docs/a.txt", "docs/archive/b.txt", "images/c.png"} {
		_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.blob.core.windows.net/assets/"+name, []byte(name), map[string]string{
			"x-ms-version":   "2023-11-03",
			"x-ms-blob-type": "BlockBlob",
		}))
	}

	listBlobs, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.blob.core.windows.net/assets?restype=container&comp=list&prefix=docs/&delimiter=/", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list blobs returned error: %v", err)
	}
	body := string(listBlobs.RawBody)
	if !strings.Contains(body, "<Prefix>docs/</Prefix>") {
		t.Fatalf("expected prefix echo in response, got: %s", body)
	}
	if !strings.Contains(body, "<Delimiter>/</Delimiter>") {
		t.Fatalf("expected delimiter echo in response, got: %s", body)
	}
	if !strings.Contains(body, "<Name>docs/a.txt</Name>") {
		t.Fatalf("expected direct child blob in response, got: %s", body)
	}
	if !strings.Contains(body, "<BlobPrefix><Name>docs/archive/</Name></BlobPrefix>") {
		t.Fatalf("expected nested blob prefix in response, got: %s", body)
	}
	if strings.Contains(body, "<Name>docs/archive/b.txt</Name>") {
		t.Fatalf("expected delimiter to hide nested blob, got: %s", body)
	}
	if strings.Contains(body, "<Name>images/c.png</Name>") {
		t.Fatalf("expected prefix to hide unrelated blob, got: %s", body)
	}
}

func TestQueueDataPlaneLifecycle(t *testing.T) {
	svc := storage.New()

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d", createQueue.StatusCode)
	}

	putMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.queue.core.windows.net/work/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>hello queue</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("put message returned error: %v", err)
	}
	if putMessage.StatusCode != http.StatusCreated {
		t.Fatalf("expected put message status 201, got %d", putMessage.StatusCode)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work/messages?numofmessages=1&visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	var messages struct {
		Messages []struct {
			MessageID  string `xml:"MessageId"`
			PopReceipt string `xml:"PopReceipt"`
			Text       string `xml:"MessageText"`
		} `xml:"QueueMessage"`
	}
	if err := xml.NewDecoder(bytes.NewReader(getMessages.RawBody)).Decode(&messages); err != nil && err != io.EOF {
		t.Fatalf("failed to decode queue messages: %v", err)
	}
	if len(messages.Messages) != 1 {
		t.Fatalf("expected one queue message, got %d in %s", len(messages.Messages), string(getMessages.RawBody))
	}
	msg := messages.Messages[0]
	if msg.Text != "hello queue" {
		t.Fatalf("unexpected queue message text: %q", msg.Text)
	}

	badDelete, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://acctest.queue.core.windows.net/work/messages/"+msg.MessageID+"?popreceipt=wrong", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("bad delete returned error: %v", err)
	}
	if badDelete.StatusCode != http.StatusNotFound {
		t.Fatalf("expected bad delete status 404, got %d", badDelete.StatusCode)
	}

	deleteMessage, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, "https://acctest.queue.core.windows.net/work/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete message returned error: %v", err)
	}
	if deleteMessage.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete message status 204, got %d", deleteMessage.StatusCode)
	}
}

func TestQueueDataPlaneMessageIDIsCanonicalGUID(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	headers := map[string]string{"x-ms-version": "2023-11-03"}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, headers))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	putMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>guid-shaped</MessageText></QueueMessage>`), headers))
	if err != nil {
		t.Fatalf("put message returned error: %v", err)
	}
	if putMessage.StatusCode != http.StatusCreated {
		t.Fatalf("expected put message status 201, got %d body=%s", putMessage.StatusCode, string(putMessage.RawBody))
	}
	created := decodeQueueMessages(t, putMessage.RawBody)
	if len(created) != 1 {
		t.Fatalf("expected put message response to contain one message, got %d body=%s", len(created), string(putMessage.RawBody))
	}
	if !looksLikeCanonicalUUID(created[0].MessageID) {
		t.Fatalf("expected put message MessageId to be a canonical GUID, got %q", created[0].MessageID)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, headers))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	dequeued := decodeQueueMessages(t, getMessages.RawBody)
	if len(dequeued) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(dequeued), string(getMessages.RawBody))
	}
	if dequeued[0].MessageID != created[0].MessageID || !looksLikeCanonicalUUID(dequeued[0].MessageID) {
		t.Fatalf("expected dequeued MessageId to preserve canonical GUID %q, got %q", created[0].MessageID, dequeued[0].MessageID)
	}
}

func TestQueuePutMessageResponseOmitsMessageTextButReceiveReturnsIt(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	headers := map[string]string{"x-ms-version": "2023-11-03"}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, headers))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	putMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>body-only-on-receive</MessageText></QueueMessage>`), headers))
	if err != nil {
		t.Fatalf("put message returned error: %v", err)
	}
	if putMessage.StatusCode != http.StatusCreated {
		t.Fatalf("expected put message status 201, got %d body=%s", putMessage.StatusCode, string(putMessage.RawBody))
	}
	putBody := string(putMessage.RawBody)
	if !strings.Contains(putBody, "<MessageId>") || !strings.Contains(putBody, "<PopReceipt>") {
		t.Fatalf("expected put message response to include message identity and receipt, got %s", putBody)
	}
	if strings.Contains(putBody, "<MessageText>") || strings.Contains(putBody, "body-only-on-receive") {
		t.Fatalf("expected put message response to omit MessageText, got %s", putBody)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, headers))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	body := string(getMessages.RawBody)
	if !strings.Contains(body, "<MessageText>body-only-on-receive</MessageText>") {
		t.Fatalf("expected get messages response to include MessageText, got %s", body)
	}
}

func TestQueuePutMessageRejectsMarkupInsideMessageText(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	headers := map[string]string{"x-ms-version": "2023-11-03"}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, headers))
	invalid, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText><sample>not encoded</sample></MessageText></QueueMessage>`), headers))
	if err != nil {
		t.Fatalf("put invalid message returned error: %v", err)
	}
	if invalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected raw markup inside MessageText to fail 400, got %d body=%s", invalid.StatusCode, string(invalid.RawBody))
	}
	if !strings.Contains(string(invalid.RawBody), "InvalidXmlDocument") {
		t.Fatalf("expected InvalidXmlDocument error, got %s", string(invalid.RawBody))
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?numofmessages=32", nil, headers))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "<QueueMessage>") {
		t.Fatalf("expected invalid message not to be queued, got %s", string(getMessages.RawBody))
	}
}

func TestQueueDataPlaneEchoesRequestedVersionHeader(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	queueURL := accountURL + "/work"
	headers := map[string]string{"x-ms-version": "2011-08-18"}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, queueURL, nil, headers))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}
	if got := createQueue.Headers["x-ms-version"]; got != "2011-08-18" {
		t.Fatalf("expected create queue to echo x-ms-version 2011-08-18, got %q", got)
	}

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list", nil, headers))
	if err != nil {
		t.Fatalf("list queues returned error: %v", err)
	}
	if listQueues.StatusCode != http.StatusOK {
		t.Fatalf("expected list queues status 200, got %d body=%s", listQueues.StatusCode, string(listQueues.RawBody))
	}
	if got := listQueues.Headers["x-ms-version"]; got != "2011-08-18" {
		t.Fatalf("expected list queues to echo x-ms-version 2011-08-18, got %q", got)
	}

	putMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPost, queueURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>versioned</MessageText></QueueMessage>`), headers))
	if err != nil {
		t.Fatalf("put message returned error: %v", err)
	}
	if putMessage.StatusCode != http.StatusCreated {
		t.Fatalf("expected put message status 201, got %d body=%s", putMessage.StatusCode, string(putMessage.RawBody))
	}
	if got := putMessage.Headers["x-ms-version"]; got != "2011-08-18" {
		t.Fatalf("expected put message to echo x-ms-version 2011-08-18, got %q", got)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queueURL+"/messages?numofmessages=1", nil, headers))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	if getMessages.StatusCode != http.StatusOK {
		t.Fatalf("expected get messages status 200, got %d body=%s", getMessages.StatusCode, string(getMessages.RawBody))
	}
	if got := getMessages.Headers["x-ms-version"]; got != "2011-08-18" {
		t.Fatalf("expected get messages to echo x-ms-version 2011-08-18, got %q", got)
	}
}

func TestQueueDataPlaneOmitsVersionHeaderBefore20090919(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	queueURL := accountURL + "/legacy"
	headers := map[string]string{"x-ms-version": "2009-07-17"}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, queueURL, nil, headers))
	if err != nil {
		t.Fatalf("legacy create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected legacy create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}
	if _, ok := createQueue.Headers["x-ms-version"]; ok {
		t.Fatalf("expected legacy create queue not to return x-ms-version, got headers=%v", createQueue.Headers)
	}

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list", nil, headers))
	if err != nil {
		t.Fatalf("legacy list queues returned error: %v", err)
	}
	if _, ok := listQueues.Headers["x-ms-version"]; ok {
		t.Fatalf("expected legacy list queues not to return x-ms-version, got headers=%v", listQueues.Headers)
	}

	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, queueURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy version</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queueURL+"/messages?numofmessages=1", nil, headers))
	if err != nil {
		t.Fatalf("legacy get messages returned error: %v", err)
	}
	if _, ok := getMessages.Headers["x-ms-version"]; ok {
		t.Fatalf("expected legacy get messages not to return x-ms-version, got headers=%v", getMessages.Headers)
	}
}

func TestQueueDataPlaneIncludesRequestID(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	createRequestID := createQueue.Headers["x-ms-request-id"]
	if createRequestID == "" {
		t.Fatalf("expected create queue to include x-ms-request-id, got headers=%v", createQueue.Headers)
	}

	badList, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&maxresults=0", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("invalid list queues returned error: %v", err)
	}
	if badList.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid list queues status 400, got %d", badList.StatusCode)
	}
	errorRequestID := badList.Headers["x-ms-request-id"]
	if errorRequestID == "" {
		t.Fatalf("expected list queues error to include x-ms-request-id, got headers=%v", badList.Headers)
	}
	if errorRequestID == createRequestID {
		t.Fatalf("expected each queue response to get a distinct x-ms-request-id, got %q", errorRequestID)
	}
}

func TestQueueDataPlaneEchoesValidClientRequestID(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	queueURL := accountURL + "/work"
	headers := map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "queue-client-123",
	}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, queueURL, nil, headers))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.Headers["x-ms-client-request-id"] != "queue-client-123" {
		t.Fatalf("expected create queue to echo client request id, got headers=%v", createQueue.Headers)
	}

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list", nil, headers))
	if err != nil {
		t.Fatalf("list queues returned error: %v", err)
	}
	if listQueues.Headers["x-ms-client-request-id"] != "queue-client-123" {
		t.Fatalf("expected list queues to echo client request id, got headers=%v", listQueues.Headers)
	}

	putMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPost, queueURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>client request id</MessageText></QueueMessage>`), headers))
	if err != nil {
		t.Fatalf("put message returned error: %v", err)
	}
	if putMessage.Headers["x-ms-client-request-id"] != "queue-client-123" {
		t.Fatalf("expected put message to echo client request id, got headers=%v", putMessage.Headers)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, queueURL+"/messages?numofmessages=1", nil, headers))
	if err != nil {
		t.Fatalf("get messages returned error: %v", err)
	}
	if getMessages.Headers["x-ms-client-request-id"] != "queue-client-123" {
		t.Fatalf("expected get messages to echo client request id, got headers=%v", getMessages.Headers)
	}
}

func TestQueueDataPlaneSuppressesInvalidClientRequestID(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	queueURL := accountURL + "/work"

	oversizedID := strings.Repeat("a", 1025)
	oversized, err := svc.HandleRequest(storageCtx(t, http.MethodPut, queueURL, nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": oversizedID,
	}))
	if err != nil {
		t.Fatalf("create queue with oversized client request id returned error: %v", err)
	}
	if _, ok := oversized.Headers["x-ms-client-request-id"]; ok {
		t.Fatalf("expected oversized client request id not to be echoed, got headers=%v", oversized.Headers)
	}

	nonVisible, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list", nil, map[string]string{
		"x-ms-version":           "2023-11-03",
		"x-ms-client-request-id": "bad\x7f",
	}))
	if err != nil {
		t.Fatalf("list queues with non-visible client request id returned error: %v", err)
	}
	if _, ok := nonVisible.Headers["x-ms-client-request-id"]; ok {
		t.Fatalf("expected non-visible client request id not to be echoed, got headers=%v", nonVisible.Headers)
	}
}

func TestQueuePutMessageValidatesVisibilityTTLAndSize(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	largeMessage := strings.Repeat("a", 64*1024+1)
	cases := []struct {
		name      string
		targetURL string
		body      []byte
		parameter string
		value     string
		minimum   string
		maximum   string
	}{
		{
			name:      "negative visibility timeout",
			targetURL: baseURL + "/messages?visibilitytimeout=-1",
			body:      []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`),
			parameter: "visibilitytimeout",
			value:     "-1",
			minimum:   "0",
			maximum:   "604800",
		},
		{
			name:      "visibility timeout above seven days",
			targetURL: baseURL + "/messages?visibilitytimeout=604801",
			body:      []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`),
			parameter: "visibilitytimeout",
			value:     "604801",
			minimum:   "0",
			maximum:   "604800",
		},
		{
			name:      "zero message ttl",
			targetURL: baseURL + "/messages?messagettl=0",
			body:      []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`),
		},
		{
			name:      "visibility timeout not smaller than ttl",
			targetURL: baseURL + "/messages?visibilitytimeout=5&messagettl=5",
			body:      []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`),
		},
		{
			name:      "message too large",
			targetURL: baseURL + "/messages",
			body:      []byte(`<QueueMessage><MessageText>` + largeMessage + `</MessageText></QueueMessage>`),
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, tc.targetURL, tc.body, map[string]string{
				"x-ms-version": "2023-11-03",
			}))
			if err != nil {
				t.Fatalf("put message returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid put message to return 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
			if tc.parameter != "" {
				assertQueueOutOfRangeQueryParameterError(t, resp, tc.parameter, tc.value, tc.minimum, tc.maximum)
			}
		})
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?numofmessages=32", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after invalid puts returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "<QueueMessage>") {
		t.Fatalf("expected invalid put messages not to enqueue messages, got: %s", string(getMessages.RawBody))
	}
}

func TestQueuePutMessageRejectsInvalidIntegerQueryParameters(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	for _, tt := range []struct {
		targetURL string
		parameter string
		value     string
	}{
		{targetURL: baseURL + "/messages?visibilitytimeout=abc", parameter: "visibilitytimeout", value: "abc"},
		{targetURL: baseURL + "/messages?messagettl=forever", parameter: "messagettl", value: "forever"},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, tt.targetURL, []byte(`<QueueMessage><MessageText>invalid integer</MessageText></QueueMessage>`), map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("put message with invalid integer query returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid integer query to return 400 for %s, got %d body=%s", tt.targetURL, resp.StatusCode, string(resp.RawBody))
		}
		assertQueueInvalidQueryParameterError(t, resp, tt.parameter, tt.value)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?numofmessages=32", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after invalid integer puts returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "<QueueMessage>") {
		t.Fatalf("expected invalid integer put messages not to enqueue messages, got: %s", string(getMessages.RawBody))
	}
}

func TestQueuePutMessageAppliesLegacyVersionQueryLimits(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	cases := []struct {
		name       string
		targetURL  string
		apiVersion string
	}{
		{
			name:       "visibilitytimeout before supported version",
			targetURL:  baseURL + "/messages?visibilitytimeout=1",
			apiVersion: "2009-09-19",
		},
		{
			name:       "non-expiring ttl before supported version",
			targetURL:  baseURL + "/messages?messagettl=-1",
			apiVersion: "2017-04-17",
		},
		{
			name:       "ttl above seven days before supported version",
			targetURL:  baseURL + "/messages?messagettl=604801",
			apiVersion: "2017-04-17",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodPost, tc.targetURL, []byte(`<QueueMessage><MessageText>legacy invalid</MessageText></QueueMessage>`), map[string]string{
				"x-ms-version": tc.apiVersion,
			}))
			if err != nil {
				t.Fatalf("put message returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected legacy invalid put message to return 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
			}
		})
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after invalid legacy puts returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "legacy invalid") {
		t.Fatalf("expected invalid legacy puts not to enqueue messages, got %s", string(getMessages.RawBody))
	}
}

func TestQueuePutMessageAppliesVersionedMessageSizeLimit(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	nineKiB := strings.Repeat("a", 8*1024+1)
	legacyPut, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>`+nineKiB+`</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2009-09-19",
	}))
	if err != nil {
		t.Fatalf("legacy oversized put message returned error: %v", err)
	}
	if legacyPut.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected legacy oversized message to return 400, got %d body=%s", legacyPut.StatusCode, string(legacyPut.RawBody))
	}
	if !strings.Contains(string(legacyPut.RawBody), "MessageTooLarge") {
		t.Fatalf("expected legacy oversized message to return MessageTooLarge, got body=%s", string(legacyPut.RawBody))
	}

	afterLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after legacy oversized put returned error: %v", err)
	}
	if strings.Contains(string(afterLegacy.RawBody), nineKiB) {
		t.Fatalf("expected legacy oversized message not to be enqueued, got %s", string(afterLegacy.RawBody))
	}

	modernPut, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>`+nineKiB+`</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("modern 9 KiB put message returned error: %v", err)
	}
	if modernPut.StatusCode != http.StatusCreated {
		t.Fatalf("expected modern 9 KiB put message to return 201, got %d body=%s", modernPut.StatusCode, string(modernPut.RawBody))
	}
}

func TestQueuePutMessageResponseBodyFollowsVersion(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	legacyPut, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2011-08-18",
	}))
	if err != nil {
		t.Fatalf("legacy put message returned error: %v", err)
	}
	if legacyPut.StatusCode != http.StatusCreated {
		t.Fatalf("expected legacy put message status 201, got %d body=%s", legacyPut.StatusCode, string(legacyPut.RawBody))
	}
	if len(legacyPut.RawBody) != 0 {
		t.Fatalf("expected legacy put message response body to be empty, got %s", string(legacyPut.RawBody))
	}

	getLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get legacy message returned error: %v", err)
	}
	messages := decodeQueueMessages(t, getLegacy.RawBody)
	if len(messages) != 1 || messages[0].Text != "legacy" {
		t.Fatalf("expected legacy put to enqueue message, got messages=%+v body=%s", messages, string(getLegacy.RawBody))
	}

	modernPut, err := svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>modern</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2016-05-31",
	}))
	if err != nil {
		t.Fatalf("modern put message returned error: %v", err)
	}
	if modernPut.StatusCode != http.StatusCreated {
		t.Fatalf("expected modern put message status 201, got %d body=%s", modernPut.StatusCode, string(modernPut.RawBody))
	}
	body := string(modernPut.RawBody)
	if !strings.Contains(body, "<QueueMessagesList>") || !strings.Contains(body, "<PopReceipt>") || !strings.Contains(body, "<TimeNextVisible>") {
		t.Fatalf("expected modern put message response to include message info XML, got %s", body)
	}
}

func TestQueueDataPlaneLocalRouteMetadataAndIdempotentCreate(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))
	if err != nil {
		t.Fatalf("create local queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected first local queue create status 201, got %d; body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	recreateQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))
	if err != nil {
		t.Fatalf("recreate local queue returned error: %v", err)
	}
	if recreateQueue.StatusCode != http.StatusNoContent {
		t.Fatalf("expected existing local queue create status 204, got %d; body=%s", recreateQueue.StatusCode, string(recreateQueue.RawBody))
	}

	recreateWithDifferentMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "different",
	}))
	if err != nil {
		t.Fatalf("recreate local queue with different metadata returned error: %v", err)
	}
	if recreateWithDifferentMetadata.StatusCode != http.StatusConflict {
		t.Fatalf("expected existing local queue create with different metadata status 409, got %d; body=%s", recreateWithDifferentMetadata.StatusCode, string(recreateWithDifferentMetadata.RawBody))
	}

	getInitialMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get initial queue metadata returned error: %v", err)
	}
	if getInitialMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected initial queue metadata status 200, got %d; body=%s", getInitialMetadata.StatusCode, string(getInitialMetadata.RawBody))
	}
	if getInitialMetadata.Headers["x-ms-meta-owner"] != "sdk" || getInitialMetadata.Headers["x-ms-approximate-messages-count"] != "0" {
		t.Fatalf("expected preserved queue metadata and message count, got headers=%v", getInitialMetadata.Headers)
	}

	setMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version":      "2023-11-03",
		"x-ms-meta-owner":   "updated",
		"x-ms-meta-purpose": "compat",
	}))
	if err != nil {
		t.Fatalf("set queue metadata returned error: %v", err)
	}
	if setMetadata.StatusCode != http.StatusNoContent {
		t.Fatalf("expected set queue metadata status 204, got %d; body=%s", setMetadata.StatusCode, string(setMetadata.RawBody))
	}

	getUpdatedMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get updated queue metadata returned error: %v", err)
	}
	if getUpdatedMetadata.Headers["x-ms-meta-owner"] != "updated" ||
		getUpdatedMetadata.Headers["x-ms-meta-purpose"] != "compat" ||
		getUpdatedMetadata.Headers["x-ms-approximate-messages-count"] != "0" {
		t.Fatalf("expected updated queue metadata and message count, got headers=%v", getUpdatedMetadata.Headers)
	}
}

func TestCreateQueueRejectsInvalidNames(t *testing.T) {
	svc := storage.New()

	for _, name := range []string{"ab", "-abc", "abc-", "a--b", "UPPER", "has_underscore"} {
		t.Run(name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "http://localhost:4577/devstoreaccount1-queue/"+name, nil, map[string]string{
				"x-ms-version": "2023-11-03",
			}))
			if err != nil {
				t.Fatalf("create queue returned error: %v", err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected invalid queue name %q to return 400, got %d body=%s", name, resp.StatusCode, string(resp.RawBody))
			}
		})
	}
}

func TestQueueMetadataNamesFollowAPIVersionRules(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"

	modernInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/invalidmodern", nil, map[string]string{
		"x-ms-version":   "2023-11-03",
		"x-ms-meta-1bad": "nope",
	}))
	if err != nil {
		t.Fatalf("create queue with invalid modern metadata returned error: %v", err)
	}
	if modernInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected modern invalid metadata name to return 400, got %d body=%s", modernInvalid.StatusCode, string(modernInvalid.RawBody))
	}

	legacyInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/legacy", nil, map[string]string{
		"x-ms-version":   "2009-07-17",
		"x-ms-meta-1bad": "legacy",
	}))
	if err != nil {
		t.Fatalf("create queue with legacy metadata returned error: %v", err)
	}
	if legacyInvalid.StatusCode != http.StatusCreated {
		t.Fatalf("expected pre-2009-09-19 invalid metadata name to be accepted, got %d body=%s", legacyInvalid.StatusCode, string(legacyInvalid.RawBody))
	}

	createWork, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/work", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createWork.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createWork.StatusCode, string(createWork.RawBody))
	}

	rejectedUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/work?comp=metadata", nil, map[string]string{
		"x-ms-version":       "2023-11-03",
		"x-ms-meta-bad-name": "nope",
	}))
	if err != nil {
		t.Fatalf("set queue metadata with invalid name returned error: %v", err)
	}
	if rejectedUpdate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid set metadata name to return 400, got %d body=%s", rejectedUpdate.StatusCode, string(rejectedUpdate.RawBody))
	}

	metadataAfterReject, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"/work?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get metadata after rejected update returned error: %v", err)
	}
	if metadataAfterReject.Headers["x-ms-meta-owner"] != "sdk" {
		t.Fatalf("expected rejected metadata update to preserve existing metadata, got headers=%v", metadataAfterReject.Headers)
	}
	if _, ok := metadataAfterReject.Headers["x-ms-meta-bad-name"]; ok {
		t.Fatalf("expected rejected metadata update not to store invalid metadata, got headers=%v", metadataAfterReject.Headers)
	}
}

func TestQueueMetadataRejectsDuplicateHeaders(t *testing.T) {
	svc := storage.New()

	createDuplicate, err := svc.HandleRequest(storageCtxWithHeaders(t, http.MethodPut, "http://localhost:4577/devstoreaccount1-queue/duplicate", nil, http.Header{
		"x-ms-version":    []string{"2023-11-03"},
		"x-ms-meta-owner": []string{"sdk", "duplicate"},
	}))
	if err != nil {
		t.Fatalf("create queue with duplicate metadata returned error: %v", err)
	}
	if createDuplicate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected duplicate metadata headers to return 400, got %d body=%s", createDuplicate.StatusCode, string(createDuplicate.RawBody))
	}
}

func TestQueueDataPlaneDeleteQueueRemovesQueueState(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>to-delete</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	deleteQueue, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete queue returned error: %v", err)
	}
	if deleteQueue.StatusCode != http.StatusNoContent {
		t.Fatalf("expected delete queue status 204, got %d; body=%s", deleteQueue.StatusCode, string(deleteQueue.RawBody))
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get deleted queue metadata returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted queue metadata status 404, got %d", getMetadata.StatusCode)
	}
}

func TestQueueDataPlaneListQueuesCanIncludeMetadata(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/beta", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/alpha", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list queues returned error: %v", err)
	}
	if listQueues.StatusCode != http.StatusOK {
		t.Fatalf("expected list queues status 200, got %d; body=%s", listQueues.StatusCode, string(listQueues.RawBody))
	}
	body := string(listQueues.RawBody)
	if !strings.Contains(body, `<Name>alpha</Name>`) || !strings.Contains(body, `<Name>beta</Name>`) {
		t.Fatalf("expected listed queues in response, got: %s", body)
	}
	if !strings.Contains(body, `<Metadata>`) || !strings.Contains(body, `<owner>sdk</owner>`) {
		t.Fatalf("expected queue metadata in response, got: %s", body)
	}
	if strings.Index(body, `<Name>alpha</Name>`) > strings.Index(body, `<Name>beta</Name>`) {
		t.Fatalf("expected queues to be listed alphabetically, got: %s", body)
	}
}

func TestQueueDataPlaneListQueuesReportsLegacyInvalidMetadataNames(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/legacy", nil, map[string]string{
		"x-ms-version":   "2009-07-17",
		"x-ms-meta-1bad": "legacy",
	}))

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list queues with legacy invalid metadata returned error: %v", err)
	}
	if listQueues.StatusCode != http.StatusOK {
		t.Fatalf("expected list queues with legacy invalid metadata status 200, got %d body=%s", listQueues.StatusCode, string(listQueues.RawBody))
	}
	body := string(listQueues.RawBody)
	if !strings.Contains(body, `<Name>legacy</Name>`) || !strings.Contains(body, `<x-ms-invalid-name>1bad</x-ms-invalid-name>`) {
		t.Fatalf("expected legacy invalid metadata name projection, got: %s", body)
	}
	if strings.Contains(body, `<1bad>`) {
		t.Fatalf("expected invalid metadata name not to be emitted as an XML element name, got: %s", body)
	}
}

func TestQueueDataPlaneListQueuesUsesLegacyShapeBefore20130815(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/alpha", nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&include=metadata", nil, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("legacy list queues returned error: %v", err)
	}
	if listQueues.StatusCode != http.StatusOK {
		t.Fatalf("expected legacy list queues status 200, got %d; body=%s", listQueues.StatusCode, string(listQueues.RawBody))
	}
	body := string(listQueues.RawBody)
	for _, fragment := range []string{
		`<EnumerationResults AccountName="devstoreaccount1">`,
		`<Name>alpha</Name>`,
		`<Url>https://devstoreaccount1.queue.core.windows.net/alpha</Url>`,
		`<Metadata>`,
		`<owner>sdk</owner>`,
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected legacy list queues body to contain %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "ServiceEndpoint=") {
		t.Fatalf("expected legacy list queues body to omit ServiceEndpoint attribute, got %s", body)
	}
}

func TestQueueDataPlaneListQueuesStartsNextPageAtMarker(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	for _, name := range []string{"q01", "q02", "q03", "q04", "q05"} {
		_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/"+name, nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
	}

	firstPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&prefix=q&maxresults=3", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("first list queues page returned error: %v", err)
	}
	firstBody := string(firstPage.RawBody)
	if !strings.Contains(firstBody, `<Name>q01</Name>`) ||
		!strings.Contains(firstBody, `<Name>q02</Name>`) ||
		!strings.Contains(firstBody, `<Name>q03</Name>`) ||
		!strings.Contains(firstBody, `<NextMarker>q04</NextMarker>`) ||
		strings.Contains(firstBody, `<Name>q04</Name>`) {
		t.Fatalf("expected first page to stop before q04 and return NextMarker q04, got: %s", firstBody)
	}

	secondPage, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&prefix=q&maxresults=3&marker=q04", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("second list queues page returned error: %v", err)
	}
	secondBody := string(secondPage.RawBody)
	if !strings.Contains(secondBody, `<Marker>q04</Marker>`) ||
		!strings.Contains(secondBody, `<Name>q04</Name>`) ||
		!strings.Contains(secondBody, `<Name>q05</Name>`) {
		t.Fatalf("expected second page to begin with marker q04, got: %s", secondBody)
	}
	if strings.Contains(secondBody, `<Name>q03</Name>`) || strings.Contains(secondBody, `<NextMarker>q`) {
		t.Fatalf("expected second page to omit previous queues and non-empty NextMarker, got: %s", secondBody)
	}
}

func TestQueueDataPlaneListQueuesDefaultsToFiveThousandResults(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	for i := 0; i < 5001; i++ {
		_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/"+fmt.Sprintf("q%04d", i), nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
	}

	listQueues, err := svc.HandleRequest(storageCtx(t, http.MethodGet, accountURL+"?comp=list&prefix=q", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("list queues with default maxresults returned error: %v", err)
	}
	if listQueues.StatusCode != http.StatusOK {
		t.Fatalf("expected list queues status 200, got %d body=%s", listQueues.StatusCode, string(listQueues.RawBody))
	}
	body := string(listQueues.RawBody)
	suffixStart := len(body) - 300
	if suffixStart < 0 {
		suffixStart = 0
	}
	if !strings.Contains(body, `<Name>q4999</Name>`) ||
		strings.Contains(body, `<Name>q5000</Name>`) ||
		!strings.Contains(body, `<NextMarker>q5000</NextMarker>`) {
		t.Fatalf("expected default list page to stop at 5000 queues with q5000 as NextMarker, got suffix: %s", body[suffixStart:])
	}
	if strings.Contains(body, `<MaxResults>`) {
		t.Fatalf("expected MaxResults element only when maxresults is specified, got suffix: %s", body[suffixStart:])
	}
}

func TestQueueDataPlaneListQueuesRejectsInvalidMaxResults(t *testing.T) {
	svc := storage.New()
	accountURL := "http://localhost:4577/devstoreaccount1-queue"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, accountURL+"/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	for _, targetURL := range []string{
		accountURL + "?comp=list&maxresults=0",
		accountURL + "?comp=list&maxresults=-1",
		accountURL + "?comp=list&maxresults=abc",
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, targetURL, nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("list queues with invalid maxresults returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid maxresults to return 400 for %s, got %d body=%s", targetURL, resp.StatusCode, string(resp.RawBody))
		}
		if !strings.Contains(string(resp.RawBody), "InvalidQueryParameterValue") {
			t.Fatalf("expected InvalidQueryParameterValue for %s, got body=%s", targetURL, string(resp.RawBody))
		}
	}
}

func TestQueueGetMessagesRejectsInvalidNumOfMessages(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	for _, tt := range []struct {
		targetURL string
		value     string
	}{
		{targetURL: baseURL + "/messages?numofmessages=0", value: "0"},
		{targetURL: baseURL + "/messages?numofmessages=33", value: "33"},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tt.targetURL, nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("get messages returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid numofmessages to return 400 for %s, got %d", tt.targetURL, resp.StatusCode)
		}
		assertQueueOutOfRangeQueryParameterError(t, resp, "numofmessages", tt.value, "1", "32")
	}
}

func TestQueuePeekMessagesRejectsInvalidNumOfMessages(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	for _, tt := range []struct {
		targetURL string
		value     string
	}{
		{targetURL: baseURL + "/messages?peekonly=true&numofmessages=0", value: "0"},
		{targetURL: baseURL + "/messages?peekonly=true&numofmessages=33", value: "33"},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tt.targetURL, nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("peek messages returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid peek numofmessages to return 400 for %s, got %d", tt.targetURL, resp.StatusCode)
		}
		assertQueueOutOfRangeQueryParameterError(t, resp, "numofmessages", tt.value, "1", "32")
	}
}

func assertQueueOutOfRangeQueryParameterError(t *testing.T, resp *service.Response, parameter, value, minimum, maximum string) {
	t.Helper()
	assertQueueStorageError(t, resp, "OutOfRangeQueryParameterValue")
	body := string(resp.RawBody)
	for _, want := range []string{
		"<QueryParameterName>" + parameter + "</QueryParameterName>",
		"<QueryParameterValue>" + value + "</QueryParameterValue>",
		"<MinimumAllowed>" + minimum + "</MinimumAllowed>",
		"<MaximumAllowed>" + maximum + "</MaximumAllowed>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected invalid %s error to contain %s, got %s", parameter, want, body)
		}
	}
}

func assertQueueInvalidQueryParameterError(t *testing.T, resp *service.Response, parameter, value string) {
	t.Helper()
	assertQueueStorageError(t, resp, "InvalidQueryParameterValue")
	body := string(resp.RawBody)
	for _, want := range []string{
		"<QueryParameterName>" + parameter + "</QueryParameterName>",
		"<QueryParameterValue>" + value + "</QueryParameterValue>",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected invalid %s error to contain %s, got %s", parameter, want, body)
		}
	}
}

func assertQueueMissingRequiredQueryParameterError(t *testing.T, resp *service.Response, parameter string) {
	t.Helper()
	assertQueueStorageError(t, resp, "MissingRequiredQueryParameter")
	want := "<QueryParameterName>" + parameter + "</QueryParameterName>"
	if !strings.Contains(string(resp.RawBody), want) {
		t.Fatalf("expected missing query parameter error to contain %s, got %s", want, string(resp.RawBody))
	}
}

func assertQueueStorageError(t *testing.T, resp *service.Response, code string) {
	t.Helper()
	if resp.RawContentType != "application/xml" {
		t.Fatalf("expected Azure Storage XML error content type, got %q body=%s", resp.RawContentType, string(resp.RawBody))
	}
	body := string(resp.RawBody)
	if !strings.Contains(body, "<Code>"+code+"</Code>") {
		t.Fatalf("expected Azure Storage XML error code %s, got %s", code, body)
	}
	if resp.Headers["x-ms-error-code"] != code {
		t.Fatalf("expected x-ms-error-code %s, got headers=%v", code, resp.Headers)
	}
}

func TestQueueUpdateMessageRequiresPopReceiptAndVisibilityTimeout(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	for _, tt := range []struct {
		name      string
		targetURL string
		parameter string
	}{
		{
			name:      "missing popreceipt",
			targetURL: baseURL + "/messages/" + msg.MessageID + "?visibilitytimeout=0",
			parameter: "popreceipt",
		},
		{
			name:      "missing visibilitytimeout",
			targetURL: baseURL + "/messages/" + msg.MessageID + "?popreceipt=" + msg.PopReceipt,
			parameter: "visibilitytimeout",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, tt.targetURL, []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`), map[string]string{
				"x-ms-version": "2023-11-03",
			}))
			if err != nil {
				t.Fatalf("update message missing %s returned error: %v", tt.parameter, err)
			}
			if resp.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected missing %s to return 400, got %d body=%s", tt.parameter, resp.StatusCode, string(resp.RawBody))
			}
			assertQueueMissingRequiredQueryParameterError(t, resp, tt.parameter)
		})
	}

	validUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("valid update after missing query parameter attempts returned error: %v", err)
	}
	if validUpdate.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing query parameter attempts to preserve current receipt, got valid update status %d body=%s", validUpdate.StatusCode, string(validUpdate.RawBody))
	}
}

func TestQueueGetMessagesRejectsInvalidVisibilityTimeoutFormat(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>visible</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=abc", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages with invalid visibilitytimeout returned error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid visibilitytimeout format to return 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	assertQueueInvalidQueryParameterError(t, resp, "visibilitytimeout", "abc")

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after invalid visibilitytimeout returned error: %v", err)
	}
	if !strings.Contains(string(getMessages.RawBody), "<MessageText>visible</MessageText>") {
		t.Fatalf("expected invalid visibilitytimeout not to hide message, got: %s", string(getMessages.RawBody))
	}
}

func TestQueueGetMessagesValidatesVisibilityTimeout(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>visible</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	for _, tt := range []struct {
		targetURL string
		value     string
	}{
		{targetURL: baseURL + "/messages?visibilitytimeout=0", value: "0"},
		{targetURL: baseURL + "/messages?visibilitytimeout=604801", value: "604801"},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodGet, tt.targetURL, nil, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("get messages returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid visibilitytimeout to return 400 for %s, got %d body=%s", tt.targetURL, resp.StatusCode, string(resp.RawBody))
		}
		assertQueueOutOfRangeQueryParameterError(t, resp, "visibilitytimeout", tt.value, "1", "604800")
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after invalid visibility timeouts returned error: %v", err)
	}
	if !strings.Contains(string(getMessages.RawBody), "<MessageText>visible</MessageText>") {
		t.Fatalf("expected invalid receives not to hide or remove message, got: %s", string(getMessages.RawBody))
	}
}

func TestQueueGetMessagesAppliesLegacyVisibilityTimeoutMaximum(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>visible</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	legacyReceive, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=7201", nil, map[string]string{
		"x-ms-version": "2011-08-17",
	}))
	if err != nil {
		t.Fatalf("legacy get messages returned error: %v", err)
	}
	if legacyReceive.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected legacy visibilitytimeout above two hours to return 400, got %d body=%s", legacyReceive.StatusCode, string(legacyReceive.RawBody))
	}

	modernReceive, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=7201", nil, map[string]string{
		"x-ms-version": "2011-08-18",
	}))
	if err != nil {
		t.Fatalf("modern get messages returned error: %v", err)
	}
	if modernReceive.StatusCode != http.StatusOK {
		t.Fatalf("expected modern visibilitytimeout above two hours to be accepted, got %d body=%s", modernReceive.StatusCode, string(modernReceive.RawBody))
	}
	if !strings.Contains(string(modernReceive.RawBody), "<MessageText>visible</MessageText>") {
		t.Fatalf("expected invalid legacy receive not to hide or remove message, got %s", string(modernReceive.RawBody))
	}
}

func TestQueueGetMessagesIncrementsDequeueCount(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>retryable</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	firstDequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("first dequeue returned error: %v", err)
	}
	first := decodeQueueMessages(t, firstDequeue.RawBody)
	if len(first) != 1 || first[0].DequeueCount != 1 {
		t.Fatalf("expected first dequeue count 1, got messages=%+v body=%s", first, string(firstDequeue.RawBody))
	}

	updateMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+first[0].MessageID+"?popreceipt="+first[0].PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>retryable</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("update message returned error: %v", err)
	}
	if updateMessage.StatusCode != http.StatusNoContent {
		t.Fatalf("expected update message status 204, got %d body=%s", updateMessage.StatusCode, string(updateMessage.RawBody))
	}

	secondDequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("second dequeue returned error: %v", err)
	}
	second := decodeQueueMessages(t, secondDequeue.RawBody)
	if len(second) != 1 || second[0].DequeueCount != 2 {
		t.Fatalf("expected second dequeue count 2, got messages=%+v body=%s", second, string(secondDequeue.RawBody))
	}
	if second[0].PopReceipt == first[0].PopReceipt {
		t.Fatalf("expected each dequeue to rotate pop receipt, got %q", second[0].PopReceipt)
	}
}

func TestQueuePeekMessagesReturnsDequeueCountWithoutMutating(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>peek-count</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	first := decodeQueueMessages(t, dequeue.RawBody)
	if len(first) != 1 || first[0].DequeueCount != 1 {
		t.Fatalf("expected first dequeue count 1, got messages=%+v body=%s", first, string(dequeue.RawBody))
	}

	updateMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+first[0].MessageID+"?popreceipt="+first[0].PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>peek-count</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("update message returned error: %v", err)
	}
	if updateMessage.StatusCode != http.StatusNoContent {
		t.Fatalf("expected update message status 204, got %d body=%s", updateMessage.StatusCode, string(updateMessage.RawBody))
	}

	peek, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?peekonly=true", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("peek returned error: %v", err)
	}
	body := string(peek.RawBody)
	if !strings.Contains(body, "<DequeueCount>1</DequeueCount>") {
		t.Fatalf("expected peek to return current dequeue count, got %s", body)
	}
	if strings.Contains(body, "<PopReceipt>") || strings.Contains(body, "<TimeNextVisible>") {
		t.Fatalf("expected peek response not to include pop receipt or next-visible time, got %s", body)
	}

	secondDequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("second dequeue returned error: %v", err)
	}
	second := decodeQueueMessages(t, secondDequeue.RawBody)
	if len(second) != 1 || second[0].DequeueCount != 2 {
		t.Fatalf("expected peek not to increment dequeue count before second dequeue, got messages=%+v body=%s", second, string(secondDequeue.RawBody))
	}
}

func TestQueueGetMessagesOmitsDequeueCountForPre20090919Queues(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/legacy"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2009-07-17",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("legacy queue dequeue returned error: %v", err)
	}
	body := string(dequeue.RawBody)
	if !strings.Contains(body, "<MessageText>legacy</MessageText>") {
		t.Fatalf("expected legacy queue dequeue to return message text, got %s", body)
	}
	if strings.Contains(body, "<DequeueCount>") {
		t.Fatalf("expected legacy queue dequeue to omit DequeueCount, got %s", body)
	}
}

func TestQueuePeekMessagesOmitsDequeueCountForPre20090919Queues(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/legacy"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2009-07-17",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy peek</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("legacy queue dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one legacy dequeue message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	updateMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+messages[0].MessageID+"?popreceipt="+messages[0].PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy peek</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("legacy queue update returned error: %v", err)
	}
	if updateMessage.StatusCode != http.StatusNoContent {
		t.Fatalf("expected legacy queue update status 204, got %d body=%s", updateMessage.StatusCode, string(updateMessage.RawBody))
	}

	peek, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?peekonly=true", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("legacy queue peek returned error: %v", err)
	}
	body := string(peek.RawBody)
	if !strings.Contains(body, "<MessageText>legacy peek</MessageText>") {
		t.Fatalf("expected legacy peek to return message text, got %s", body)
	}
	if strings.Contains(body, "<DequeueCount>") {
		t.Fatalf("expected legacy peek to omit DequeueCount, got %s", body)
	}
}

func TestQueueDeleteMessageRequiresCurrentPopReceipt(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>delete-me</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	missingReceiptDelete, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages/"+msg.MessageID, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete without pop receipt returned error: %v", err)
	}
	if missingReceiptDelete.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing pop receipt to return 400, got %d body=%s", missingReceiptDelete.StatusCode, string(missingReceiptDelete.RawBody))
	}
	assertQueueStorageError(t, missingReceiptDelete, "MissingRequiredQueryParameter")

	updateMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>delete-me</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("update message returned error: %v", err)
	}
	newReceipt := updateMessage.Headers["x-ms-popreceipt"]
	if updateMessage.StatusCode != http.StatusNoContent || newReceipt == "" || newReceipt == msg.PopReceipt {
		t.Fatalf("expected update to rotate pop receipt, status=%d headers=%v", updateMessage.StatusCode, updateMessage.Headers)
	}

	staleReceiptDelete, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete with stale pop receipt returned error: %v", err)
	}
	if staleReceiptDelete.StatusCode != http.StatusNotFound {
		t.Fatalf("expected stale pop receipt to return 404, got %d body=%s", staleReceiptDelete.StatusCode, string(staleReceiptDelete.RawBody))
	}

	currentReceiptDelete, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+newReceipt, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete with current pop receipt returned error: %v", err)
	}
	if currentReceiptDelete.StatusCode != http.StatusNoContent {
		t.Fatalf("expected current pop receipt delete status 204, got %d body=%s", currentReceiptDelete.StatusCode, string(currentReceiptDelete.RawBody))
	}
}

func TestQueueUpdateMessageValidatesCurrentReceiptSizeAndExpiry(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0&messagettl=5", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	largeMessage := strings.Repeat("a", 64*1024+1)
	oversizedUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>`+largeMessage+`</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("oversized update returned error: %v", err)
	}
	if oversizedUpdate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected oversized update status 400, got %d body=%s", oversizedUpdate.StatusCode, string(oversizedUpdate.RawBody))
	}
	if !strings.Contains(string(oversizedUpdate.RawBody), "MessageTooLarge") {
		t.Fatalf("expected oversized update to return MessageTooLarge, got body=%s", string(oversizedUpdate.RawBody))
	}

	tooLongVisibility, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=10", []byte(`<QueueMessage><MessageText>too-late</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("too-long visibility update returned error: %v", err)
	}
	if tooLongVisibility.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected visibility past expiry update status 400, got %d body=%s", tooLongVisibility.StatusCode, string(tooLongVisibility.RawBody))
	}

	validUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("valid update returned error: %v", err)
	}
	newReceipt := validUpdate.Headers["x-ms-popreceipt"]
	if validUpdate.StatusCode != http.StatusNoContent || newReceipt == "" || newReceipt == msg.PopReceipt {
		t.Fatalf("expected valid update to rotate receipt, status=%d headers=%v", validUpdate.StatusCode, validUpdate.Headers)
	}

	staleUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>stale</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("stale update returned error: %v", err)
	}
	if staleUpdate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected stale pop receipt update status 400, got %d body=%s", staleUpdate.StatusCode, string(staleUpdate.RawBody))
	}
	if !strings.Contains(string(staleUpdate.RawBody), "PopReceiptMismatch") {
		t.Fatalf("expected stale pop receipt update to return PopReceiptMismatch, got body=%s", string(staleUpdate.RawBody))
	}

	stillVisible, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?peekonly=true", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("peek after update validation returned error: %v", err)
	}
	body := string(stillVisible.RawBody)
	if !strings.Contains(body, "<MessageText>after</MessageText>") || strings.Contains(body, "<MessageText>stale</MessageText>") || strings.Contains(body, largeMessage) {
		t.Fatalf("expected failed updates not to mutate message, got: %s", body)
	}
}

func TestQueueUpdateMessageRejectsOutOfRangeVisibilityTimeout(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	for _, tt := range []struct {
		targetURL string
		value     string
	}{
		{targetURL: baseURL + "/messages/" + msg.MessageID + "?popreceipt=" + msg.PopReceipt + "&visibilitytimeout=-1", value: "-1"},
		{targetURL: baseURL + "/messages/" + msg.MessageID + "?popreceipt=" + msg.PopReceipt + "&visibilitytimeout=604801", value: "604801"},
	} {
		resp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, tt.targetURL, []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`), map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("out-of-range update returned error: %v", err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected out-of-range update to return 400 for %s, got %d body=%s", tt.targetURL, resp.StatusCode, string(resp.RawBody))
		}
		assertQueueOutOfRangeQueryParameterError(t, resp, "visibilitytimeout", tt.value, "0", "604800")
	}

	validUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("valid update after out-of-range attempts returned error: %v", err)
	}
	if validUpdate.StatusCode != http.StatusNoContent {
		t.Fatalf("expected invalid visibility timeout updates to preserve current receipt, got valid update status %d body=%s", validUpdate.StatusCode, string(validUpdate.RawBody))
	}
}

func TestQueueUpdateMessageRejectsInvalidVisibilityTimeoutFormat(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	resp, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=abc", []byte(`<QueueMessage><MessageText>invalid</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("update message with invalid visibilitytimeout returned error: %v", err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected invalid update visibilitytimeout format to return 400, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	assertQueueInvalidQueryParameterError(t, resp, "visibilitytimeout", "abc")

	validUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("valid update after invalid visibilitytimeout returned error: %v", err)
	}
	if validUpdate.StatusCode != http.StatusNoContent {
		t.Fatalf("expected invalid visibilitytimeout format to preserve current receipt, got valid update status %d body=%s", validUpdate.StatusCode, string(validUpdate.RawBody))
	}
}

func TestQueueUpdateMessageRejectsLegacyVersionWithoutMutating(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue returned error: %v", err)
	}
	messages := decodeQueueMessages(t, dequeue.RawBody)
	if len(messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d body=%s", len(messages), string(dequeue.RawBody))
	}
	msg := messages[0]

	legacyUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>legacy mutation</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2009-09-19",
	}))
	if err != nil {
		t.Fatalf("legacy update returned error: %v", err)
	}
	if legacyUpdate.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected legacy update to return 400, got %d body=%s", legacyUpdate.StatusCode, string(legacyUpdate.RawBody))
	}

	modernUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("modern update with original receipt returned error: %v", err)
	}
	if modernUpdate.StatusCode != http.StatusNoContent {
		t.Fatalf("expected modern update with original receipt to return 204, got %d body=%s", modernUpdate.StatusCode, string(modernUpdate.RawBody))
	}

	peek, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?peekonly=true", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("peek after legacy update returned error: %v", err)
	}
	body := string(peek.RawBody)
	if !strings.Contains(body, "<MessageText>after</MessageText>") || strings.Contains(body, "legacy mutation") {
		t.Fatalf("expected rejected legacy update not to mutate message, got %s", body)
	}
}

func TestQueueExpiredMessagesArePrunedFromReadsAndMetadata(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages?messagettl=1", []byte(`<QueueMessage><MessageText>short lived</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	beforeExpiry, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get metadata before expiry returned error: %v", err)
	}
	if beforeExpiry.Headers["x-ms-approximate-messages-count"] != "1" {
		t.Fatalf("expected one queued message before expiry, got headers=%v", beforeExpiry.Headers)
	}

	time.Sleep(1100 * time.Millisecond)

	afterExpiry, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get metadata after expiry returned error: %v", err)
	}
	if afterExpiry.Headers["x-ms-approximate-messages-count"] != "0" {
		t.Fatalf("expected expired message to be removed from metadata count, got headers=%v", afterExpiry.Headers)
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after expiry returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "<QueueMessage>") {
		t.Fatalf("expected expired message not to be returned, got: %s", string(getMessages.RawBody))
	}
}

func TestQueueMetadataSupportsHead(t *testing.T) {
	svc := storage.New()
	baseURL := "https://acctest.queue.core.windows.net/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version":    "2023-11-03",
		"x-ms-meta-owner": "sdk",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>head count</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	headMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?comp=metadata", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head queue metadata returned error: %v", err)
	}
	if headMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected head queue metadata status 200, got %d body=%s", headMetadata.StatusCode, string(headMetadata.RawBody))
	}
	if len(headMetadata.RawBody) != 0 {
		t.Fatalf("expected head queue metadata to return no body, got %q", string(headMetadata.RawBody))
	}
	if headMetadata.Headers["x-ms-meta-owner"] != "sdk" || headMetadata.Headers["x-ms-approximate-messages-count"] != "1" {
		t.Fatalf("expected queue metadata and approximate message count headers, got %v", headMetadata.Headers)
	}
}

func TestQueueACLSetGetHeadReplaceAndLimit(t *testing.T) {
	svc := storage.New()
	baseURL := "https://acctest.queue.core.windows.net/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	firstACL := []byte(`<SignedIdentifiers>
  <SignedIdentifier>
    <Id>read-policy</Id>
    <AccessPolicy>
      <Start>2026-06-16T00:00:00.0000000Z</Start>
      <Expiry>2026-06-17T00:00:00.0000000Z</Expiry>
      <Permission>r</Permission>
    </AccessPolicy>
  </SignedIdentifier>
  <SignedIdentifier>
    <Id>process-policy</Id>
    <AccessPolicy>
      <Start>2026-06-16T00:00:00.0000000Z</Start>
      <Expiry>2026-06-17T00:00:00.0000000Z</Expiry>
      <Permission>raup</Permission>
    </AccessPolicy>
  </SignedIdentifier>
</SignedIdentifiers>`)
	setACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=acl", firstACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue ACL returned error: %v", err)
	}
	if setACL.StatusCode != http.StatusNoContent || len(setACL.RawBody) != 0 || setACL.Headers["x-ms-version"] == "" {
		t.Fatalf("expected set queue ACL status 204 without body and version header, got status=%d headers=%v body=%s", setACL.StatusCode, setACL.Headers, string(setACL.RawBody))
	}

	getACL, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue ACL returned error: %v", err)
	}
	if getACL.StatusCode != http.StatusOK || getACL.RawContentType != "application/xml" {
		t.Fatalf("expected get queue ACL status 200 XML, got status=%d contentType=%q body=%s", getACL.StatusCode, getACL.RawContentType, string(getACL.RawBody))
	}
	body := string(getACL.RawBody)
	for _, fragment := range []string{
		"<SignedIdentifiers>",
		"<Id>read-policy</Id>",
		"<Permission>r</Permission>",
		"<Id>process-policy</Id>",
		"<Permission>raup</Permission>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected get queue ACL body to contain %q, got %s", fragment, body)
		}
	}

	headACL, err := svc.HandleRequest(storageCtx(t, http.MethodHead, baseURL+"?comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("head queue ACL returned error: %v", err)
	}
	if headACL.StatusCode != http.StatusOK || len(headACL.RawBody) != 0 || headACL.RawContentType != "application/xml" {
		t.Fatalf("expected head queue ACL status 200 XML without body, got status=%d contentType=%q body=%s", headACL.StatusCode, headACL.RawContentType, string(headACL.RawBody))
	}

	replacementACL := []byte(`<SignedIdentifiers><SignedIdentifier><Id>replace-policy</Id><AccessPolicy><Permission>p</Permission></AccessPolicy></SignedIdentifier></SignedIdentifiers>`)
	replaceACL, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=acl", replacementACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("replace queue ACL returned error: %v", err)
	}
	if replaceACL.StatusCode != http.StatusNoContent {
		t.Fatalf("expected replace queue ACL status 204, got %d body=%s", replaceACL.StatusCode, string(replaceACL.RawBody))
	}

	replaced, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get replaced queue ACL returned error: %v", err)
	}
	replacedBody := string(replaced.RawBody)
	if !strings.Contains(replacedBody, "<Id>replace-policy</Id>") || strings.Contains(replacedBody, "read-policy") {
		t.Fatalf("expected set queue ACL to replace existing policies, got %s", replacedBody)
	}

	tooManyACL := []byte(`<SignedIdentifiers>
<SignedIdentifier><Id>p1</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p2</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p3</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p4</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p5</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
<SignedIdentifier><Id>p6</Id><AccessPolicy><Permission>r</Permission></AccessPolicy></SignedIdentifier>
</SignedIdentifiers>`)
	setTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"?comp=acl", tooManyACL, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too many queue ACL policies returned error: %v", err)
	}
	if setTooMany.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many queue ACL policies to return 400, got %d body=%s", setTooMany.StatusCode, string(setTooMany.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"?comp=acl", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue ACL after invalid update returned error: %v", err)
	}
	afterInvalidBody := string(afterInvalid.RawBody)
	if !strings.Contains(afterInvalidBody, "<Id>replace-policy</Id>") || strings.Contains(afterInvalidBody, "<Id>p6</Id>") {
		t.Fatalf("expected invalid queue ACL update not to replace existing policies, got %s", afterInvalidBody)
	}
}

func TestQueueServicePropertiesGetSetRoundTrip(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	defaultProperties, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get default queue service properties returned error: %v", err)
	}
	if defaultProperties.StatusCode != http.StatusOK || defaultProperties.RawContentType != "application/xml" {
		t.Fatalf("expected default queue service properties status 200 XML, got status=%d contentType=%q body=%s", defaultProperties.StatusCode, defaultProperties.RawContentType, string(defaultProperties.RawBody))
	}
	defaultBody := string(defaultProperties.RawBody)
	for _, fragment := range []string{"<StorageServiceProperties>", "<Logging>", "<HourMetrics>", "<MinuteMetrics>", "<Cors>"} {
		if !strings.Contains(defaultBody, fragment) {
			t.Fatalf("expected default queue service properties to include %q, got %s", fragment, defaultBody)
		}
	}

	customProperties := []byte(`<StorageServiceProperties>
  <HourMetrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>true</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>7</Days></RetentionPolicy>
  </HourMetrics>
  <Cors>
    <CorsRule>
      <AllowedOrigins>https://example.com</AllowedOrigins>
      <AllowedMethods>GET,PUT</AllowedMethods>
      <MaxAgeInSeconds>500</MaxAgeInSeconds>
      <ExposedHeaders>x-ms-meta-data*</ExposedHeaders>
      <AllowedHeaders>x-ms-meta-target*</AllowedHeaders>
    </CorsRule>
  </Cors>
</StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, customProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted || len(setProperties.RawBody) != 0 || setProperties.Headers["x-ms-version"] == "" {
		t.Fatalf("expected set queue service properties status 202 without body and version header, got status=%d headers=%v body=%s", setProperties.StatusCode, setProperties.Headers, string(setProperties.RawBody))
	}

	updatedProperties, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get updated queue service properties returned error: %v", err)
	}
	updatedBody := string(updatedProperties.RawBody)
	for _, fragment := range []string{
		"<AllowedOrigins>https://example.com</AllowedOrigins>",
		"<AllowedMethods>GET,PUT</AllowedMethods>",
		"<IncludeAPIs>true</IncludeAPIs>",
		"<Days>7</Days>",
	} {
		if !strings.Contains(updatedBody, fragment) {
			t.Fatalf("expected updated queue service properties to include %q, got %s", fragment, updatedBody)
		}
	}
}

func TestQueueServicePropertiesPreservesOmittedRootElements(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	initialProperties := []byte(`<StorageServiceProperties>
  <HourMetrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>true</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>7</Days></RetentionPolicy>
  </HourMetrics>
  <Cors>
    <CorsRule>
      <AllowedOrigins>https://old.example</AllowedOrigins>
      <AllowedMethods>GET</AllowedMethods>
      <MaxAgeInSeconds>60</MaxAgeInSeconds>
      <ExposedHeaders>x-ms-meta-data*</ExposedHeaders>
      <AllowedHeaders>x-ms-meta-target*</AllowedHeaders>
    </CorsRule>
  </Cors>
</StorageServiceProperties>`)
	setInitial, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, initialProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set initial queue service properties returned error: %v", err)
	}
	if setInitial.StatusCode != http.StatusAccepted {
		t.Fatalf("expected initial queue service properties status 202, got %d body=%s", setInitial.StatusCode, string(setInitial.RawBody))
	}

	corsOnlyUpdate := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://new.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>120</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-new*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setCorsOnly, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, corsOnlyUpdate, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set CORS-only queue service properties returned error: %v", err)
	}
	if setCorsOnly.StatusCode != http.StatusAccepted {
		t.Fatalf("expected CORS-only queue service properties status 202, got %d body=%s", setCorsOnly.StatusCode, string(setCorsOnly.RawBody))
	}

	afterUpdate, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after CORS-only update returned error: %v", err)
	}
	body := string(afterUpdate.RawBody)
	for _, fragment := range []string{
		"<IncludeAPIs>true</IncludeAPIs>",
		"<Days>7</Days>",
		"https://new.example",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected partial Queue service property update to preserve/include %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "https://old.example") {
		t.Fatalf("expected CORS-only update to replace prior CORS rules, got %s", body)
	}
}

func TestQueueServicePropertiesGetProjectsLegacyMetricsForOlderVersions(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	modernProperties := []byte(`<StorageServiceProperties>
  <HourMetrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>true</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>7</Days></RetentionPolicy>
  </HourMetrics>
  <MinuteMetrics>
    <Version>1.0</Version>
    <Enabled>false</Enabled>
    <RetentionPolicy><Enabled>false</Enabled></RetentionPolicy>
  </MinuteMetrics>
  <Cors>
    <CorsRule>
      <AllowedOrigins>https://modern.example</AllowedOrigins>
      <AllowedMethods>GET</AllowedMethods>
      <MaxAgeInSeconds>60</MaxAgeInSeconds>
      <ExposedHeaders>x-ms-meta-data*</ExposedHeaders>
      <AllowedHeaders>x-ms-meta-target*</AllowedHeaders>
    </CorsRule>
  </Cors>
</StorageServiceProperties>`)
	setModern, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, modernProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern queue service properties returned error: %v", err)
	}
	if setModern.StatusCode != http.StatusAccepted {
		t.Fatalf("expected modern queue service properties status 202, got %d body=%s", setModern.StatusCode, string(setModern.RawBody))
	}

	legacy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("get legacy queue service properties returned error: %v", err)
	}
	if legacy.StatusCode != http.StatusOK {
		t.Fatalf("expected legacy queue service properties status 200, got %d body=%s", legacy.StatusCode, string(legacy.RawBody))
	}
	body := string(legacy.RawBody)
	for _, fragment := range []string{
		"<Metrics>",
		"<IncludeAPIs>true</IncludeAPIs>",
		"<Days>7</Days>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected legacy queue service properties to include %q, got %s", fragment, body)
		}
	}
	for _, fragment := range []string{"<HourMetrics>", "<MinuteMetrics>", "<Cors>", "https://modern.example"} {
		if strings.Contains(body, fragment) {
			t.Fatalf("expected legacy queue service properties to omit %q, got %s", fragment, body)
		}
	}
}

func TestQueueServicePropertiesRejectsModernRootsForLegacyVersions(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	modernProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setModern, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, modernProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern queue service properties returned error: %v", err)
	}
	if setModern.StatusCode != http.StatusAccepted {
		t.Fatalf("expected modern queue service properties status 202, got %d body=%s", setModern.StatusCode, string(setModern.RawBody))
	}

	legacyCors := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://legacy.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setLegacyCors, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, legacyCors, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("set legacy queue service properties with CORS returned error: %v", err)
	}
	if setLegacyCors.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected legacy queue service properties with CORS status 400, got %d body=%s", setLegacyCors.StatusCode, string(setLegacyCors.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after legacy CORS update returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://legacy.example") {
		t.Fatalf("expected legacy CORS update not to replace modern properties, got %s", body)
	}
}

func TestQueueServicePropertiesModernGetAfterLegacySetUsesModernMetricsShape(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	legacyProperties := []byte(`<StorageServiceProperties>
  <Logging>
    <Version>1.0</Version>
    <Delete>true</Delete>
    <Read>false</Read>
    <Write>true</Write>
    <RetentionPolicy><Enabled>true</Enabled><Days>3</Days></RetentionPolicy>
  </Logging>
  <Metrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>false</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>9</Days></RetentionPolicy>
  </Metrics>
</StorageServiceProperties>`)
	setLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, legacyProperties, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("set legacy queue service properties returned error: %v", err)
	}
	if setLegacy.StatusCode != http.StatusAccepted {
		t.Fatalf("expected legacy queue service properties status 202, got %d body=%s", setLegacy.StatusCode, string(setLegacy.RawBody))
	}

	modern, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get modern queue service properties after legacy set returned error: %v", err)
	}
	if modern.StatusCode != http.StatusOK {
		t.Fatalf("expected modern queue service properties status 200, got %d body=%s", modern.StatusCode, string(modern.RawBody))
	}
	body := string(modern.RawBody)
	for _, fragment := range []string{
		"<Logging>",
		"<HourMetrics>",
		"<MinuteMetrics>",
		"<Cors>",
		"<Delete>true</Delete>",
		"<Write>true</Write>",
		"<IncludeAPIs>false</IncludeAPIs>",
		"<Days>9</Days>",
	} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected modern queue service properties after legacy set to include %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "<Metrics>") {
		t.Fatalf("expected modern queue service properties after legacy set not to include legacy Metrics root, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsLegacyMetricsRootForModernVersions(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	modernProperties := []byte(`<StorageServiceProperties><HourMetrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>true</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>5</Days></RetentionPolicy></HourMetrics></StorageServiceProperties>`)
	setModern, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, modernProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern queue service properties returned error: %v", err)
	}
	if setModern.StatusCode != http.StatusAccepted {
		t.Fatalf("expected modern queue service properties status 202, got %d body=%s", setModern.StatusCode, string(setModern.RawBody))
	}

	legacyMetrics := []byte(`<StorageServiceProperties><Metrics><Version>1.0</Version><Enabled>true</Enabled><IncludeAPIs>true</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>11</Days></RetentionPolicy></Metrics></StorageServiceProperties>`)
	setLegacyMetrics, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, legacyMetrics, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set modern queue service properties with legacy Metrics returned error: %v", err)
	}
	if setLegacyMetrics.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected modern queue service properties with legacy Metrics status 400, got %d body=%s", setLegacyMetrics.StatusCode, string(setLegacyMetrics.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after modern legacy-Metrics update returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "<Days>5</Days>") || strings.Contains(body, "<Days>11</Days>") || strings.Contains(body, "<Metrics>") {
		t.Fatalf("expected modern legacy-Metrics update not to replace stored modern properties, got %s", body)
	}
}

func TestQueueServicePropertiesEchoesRequestedVersionHeader(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	legacyProperties := []byte(`<StorageServiceProperties>
  <Logging>
    <Version>1.0</Version>
    <Delete>true</Delete>
    <Read>false</Read>
    <Write>true</Write>
    <RetentionPolicy><Enabled>true</Enabled><Days>3</Days></RetentionPolicy>
  </Logging>
  <Metrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>false</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>9</Days></RetentionPolicy>
  </Metrics>
</StorageServiceProperties>`)
	setLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, legacyProperties, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("set legacy queue service properties returned error: %v", err)
	}
	if setLegacy.StatusCode != http.StatusAccepted {
		t.Fatalf("expected legacy queue service properties status 202, got %d body=%s", setLegacy.StatusCode, string(setLegacy.RawBody))
	}
	if got := setLegacy.Headers["x-ms-version"]; got != "2012-02-12" {
		t.Fatalf("expected set queue service properties to echo x-ms-version 2012-02-12, got %q", got)
	}

	getLegacy, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2012-02-12",
	}))
	if err != nil {
		t.Fatalf("get legacy queue service properties returned error: %v", err)
	}
	if getLegacy.StatusCode != http.StatusOK {
		t.Fatalf("expected legacy queue service properties status 200, got %d body=%s", getLegacy.StatusCode, string(getLegacy.RawBody))
	}
	if got := getLegacy.Headers["x-ms-version"]; got != "2012-02-12" {
		t.Fatalf("expected get queue service properties to echo x-ms-version 2012-02-12, got %q", got)
	}
}

func TestQueueServicePropertiesRejectsInvalidAnalyticsProperties(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"

	validProperties := []byte(`<StorageServiceProperties>
  <HourMetrics>
    <Version>1.0</Version>
    <Enabled>true</Enabled>
    <IncludeAPIs>true</IncludeAPIs>
    <RetentionPolicy><Enabled>true</Enabled><Days>7</Days></RetentionPolicy>
  </HourMetrics>
  <Cors>
    <CorsRule>
      <AllowedOrigins>https://valid.example</AllowedOrigins>
      <AllowedMethods>GET</AllowedMethods>
      <MaxAgeInSeconds>60</MaxAgeInSeconds>
      <ExposedHeaders>x-ms-meta-data*</ExposedHeaders>
      <AllowedHeaders>x-ms-meta-target*</AllowedHeaders>
    </CorsRule>
  </Cors>
</StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	invalidPayloads := []struct {
		name string
		body []byte
	}{
		{
			name: "enabled HourMetrics missing IncludeAPIs",
			body: []byte(`<StorageServiceProperties><HourMetrics><Version>9.9</Version><Enabled>true</Enabled><RetentionPolicy><Enabled>false</Enabled></RetentionPolicy></HourMetrics></StorageServiceProperties>`),
		},
		{
			name: "enabled HourMetrics retention missing Days",
			body: []byte(`<StorageServiceProperties><HourMetrics><Version>9.9</Version><Enabled>true</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled></RetentionPolicy></HourMetrics></StorageServiceProperties>`),
		},
		{
			name: "HourMetrics retention Days above range",
			body: []byte(`<StorageServiceProperties><HourMetrics><Version>9.9</Version><Enabled>true</Enabled><IncludeAPIs>false</IncludeAPIs><RetentionPolicy><Enabled>true</Enabled><Days>366</Days></RetentionPolicy></HourMetrics></StorageServiceProperties>`),
		},
	}
	for _, invalid := range invalidPayloads {
		setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, invalid.body, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("set invalid queue analytics properties %q returned error: %v", invalid.name, err)
		}
		if setInvalid.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid queue analytics properties %q status 400, got %d body=%s", invalid.name, setInvalid.StatusCode, string(setInvalid.RawBody))
		}
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after invalid analytics properties returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	for _, fragment := range []string{"<IncludeAPIs>true</IncludeAPIs>", "<Days>7</Days>", "https://valid.example"} {
		if !strings.Contains(body, fragment) {
			t.Fatalf("expected invalid analytics updates to preserve %q, got %s", fragment, body)
		}
	}
	if strings.Contains(body, "<Version>9.9</Version>") {
		t.Fatalf("expected invalid analytics updates not to replace stored HourMetrics, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsInvalidCorsRules(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	tooManyCors := []byte(`<StorageServiceProperties><Cors>
<CorsRule><AllowedOrigins>https://one.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://two.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://three.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://four.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://five.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
<CorsRule><AllowedOrigins>https://six.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule>
</Cors></StorageServiceProperties>`)
	setTooMany, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooManyCors, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too many queue service CORS rules returned error: %v", err)
	}
	if setTooMany.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many queue service CORS rules status 400, got %d body=%s", setTooMany.StatusCode, string(setTooMany.RawBody))
	}

	missingCorsField := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://missing.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setMissingField, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, missingCorsField, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set missing-field queue service CORS rule returned error: %v", err)
	}
	if setMissingField.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing CORS field status 400, got %d body=%s", setMissingField.StatusCode, string(setMissingField.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after invalid CORS updates returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://six.example") || strings.Contains(body, "https://missing.example") {
		t.Fatalf("expected invalid CORS updates not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsUnsupportedCorsMethods(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET,PUT</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	unsupportedMethod := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://patch.example</AllowedOrigins><AllowedMethods>GET,PATCH</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setUnsupported, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, unsupportedMethod, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set unsupported-method queue service CORS rule returned error: %v", err)
	}
	if setUnsupported.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected unsupported CORS method status 400, got %d body=%s", setUnsupported.StatusCode, string(setUnsupported.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after unsupported CORS method returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://patch.example") {
		t.Fatalf("expected unsupported CORS method not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsTooManyCorsAllowedOrigins(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	origins := make([]string, 65)
	for i := range origins {
		origins[i] = "https://origin" + strconv.Itoa(i) + ".example"
	}
	tooManyOrigins := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>` + strings.Join(origins, ",") + `</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooManyOrigins, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too-many-origins queue service CORS rule returned error: %v", err)
	}
	if setInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many CORS origins status 400, got %d body=%s", setInvalid.StatusCode, string(setInvalid.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after too many CORS origins returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://origin64.example") {
		t.Fatalf("expected too many CORS origins not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsTooLongCorsAllowedOrigin(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	longOrigin := "https://" + strings.Repeat("a", 245) + ".example"
	tooLongOrigin := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>` + longOrigin + `</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooLongOrigin, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too-long-origin queue service CORS rule returned error: %v", err)
	}
	if setInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too long CORS origin status 400, got %d body=%s", setInvalid.StatusCode, string(setInvalid.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after too long CORS origin returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, longOrigin) {
		t.Fatalf("expected too long CORS origin not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsTooManyCorsAllowedHeaders(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	headers := make([]string, 65)
	for i := range headers {
		headers[i] = "x-ms-meta-header-" + strconv.Itoa(i)
	}
	tooManyHeaders := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://headers.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>` + strings.Join(headers, ",") + `</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooManyHeaders, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too-many-allowed-headers queue service CORS rule returned error: %v", err)
	}
	if setInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many CORS allowed headers status 400, got %d body=%s", setInvalid.StatusCode, string(setInvalid.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after too many CORS allowed headers returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://headers.example") {
		t.Fatalf("expected too many CORS allowed headers not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsTooManyCorsExposedHeaderPrefixes(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	tooManyPrefixedHeaders := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://exposed.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*,x-ms-meta-target*,x-ms-meta-extra*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, tooManyPrefixedHeaders, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set too-many-exposed-header-prefixes queue service CORS rule returned error: %v", err)
	}
	if setInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected too many CORS exposed header prefixes status 400, got %d body=%s", setInvalid.StatusCode, string(setInvalid.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after too many CORS exposed header prefixes returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://exposed.example") {
		t.Fatalf("expected too many CORS exposed header prefixes not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsInvalidCorsMaxAgeInSeconds(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	invalidMaxAgeValues := []string{"soon", "-1"}
	for _, invalidMaxAge := range invalidMaxAgeValues {
		invalidProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://invalid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>` + invalidMaxAge + `</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
		setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, invalidProperties, map[string]string{
			"x-ms-version": "2023-11-03",
		}))
		if err != nil {
			t.Fatalf("set invalid MaxAgeInSeconds %q returned error: %v", invalidMaxAge, err)
		}
		if setInvalid.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected invalid MaxAgeInSeconds %q status 400, got %d body=%s", invalidMaxAge, setInvalid.StatusCode, string(setInvalid.RawBody))
		}
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after invalid MaxAgeInSeconds returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://invalid.example") {
		t.Fatalf("expected invalid MaxAgeInSeconds not to replace valid properties, got %s", body)
	}
}

func TestQueueServicePropertiesRejectsOversizedCorsSettings(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	validProperties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://valid.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setValid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, validProperties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set valid queue service properties returned error: %v", err)
	}
	if setValid.StatusCode != http.StatusAccepted {
		t.Fatalf("expected valid queue service properties status 202, got %d body=%s", setValid.StatusCode, string(setValid.RawBody))
	}

	headers := make([]string, 64)
	for i := range headers {
		headers[i] = "x-ms-meta-" + strings.Repeat("h", 24) + strconv.Itoa(i)
	}
	oversizedCors := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://oversized.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>60</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>` + strings.Join(headers, ",") + `</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, oversizedCors, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set oversized queue service CORS settings returned error: %v", err)
	}
	if setInvalid.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected oversized CORS settings status 400, got %d body=%s", setInvalid.StatusCode, string(setInvalid.RawBody))
	}

	afterInvalid, err := svc.HandleRequest(storageCtx(t, http.MethodGet, serviceURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue service properties after oversized CORS settings returned error: %v", err)
	}
	body := string(afterInvalid.RawBody)
	if !strings.Contains(body, "https://valid.example") || strings.Contains(body, "https://oversized.example") {
		t.Fatalf("expected oversized CORS settings not to replace valid properties, got %s", body)
	}
}

func TestQueueCORSPreflightAllowsMatchingStoredRule(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>PUT,GET</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*,x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	preflight, err := svc.HandleRequest(storageCtx(t, http.MethodOptions, "https://acctest.queue.core.windows.net/work/messages", nil, map[string]string{
		"Origin":                         "https://app.example",
		"Access-Control-Request-Method":  "PUT",
		"Access-Control-Request-Headers": "x-ms-meta-target-id,x-ms-client-request-id",
		"x-ms-version":                   "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("queue CORS preflight returned error: %v", err)
	}
	if preflight.StatusCode != http.StatusOK {
		t.Fatalf("expected queue CORS preflight status 200, got %d body=%s", preflight.StatusCode, string(preflight.RawBody))
	}
	if preflight.Headers["Access-Control-Allow-Origin"] != "https://app.example" {
		t.Fatalf("expected allow-origin header for matching origin, got %q", preflight.Headers["Access-Control-Allow-Origin"])
	}
	if preflight.Headers["Access-Control-Allow-Methods"] != "PUT" {
		t.Fatalf("expected allow-methods header for requested method, got %q", preflight.Headers["Access-Control-Allow-Methods"])
	}
	if preflight.Headers["Access-Control-Allow-Headers"] != "x-ms-meta-target-id,x-ms-client-request-id" {
		t.Fatalf("expected allow-headers header for requested headers, got %q", preflight.Headers["Access-Control-Allow-Headers"])
	}
	if preflight.Headers["Access-Control-Max-Age"] != "200" {
		t.Fatalf("expected max-age header from CORS rule, got %q", preflight.Headers["Access-Control-Max-Age"])
	}
}

func TestQueueCORSPreflightRejectsUnmatchedOrigin(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>PUT</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-meta-target*</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	preflight, err := svc.HandleRequest(storageCtx(t, http.MethodOptions, "https://acctest.queue.core.windows.net/work/messages", nil, map[string]string{
		"Origin":                         "https://other.example",
		"Access-Control-Request-Method":  "PUT",
		"Access-Control-Request-Headers": "x-ms-meta-target-id",
		"x-ms-version":                   "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("queue CORS preflight returned error: %v", err)
	}
	if preflight.StatusCode != http.StatusForbidden {
		t.Fatalf("expected unmatched queue CORS preflight status 403, got %d body=%s", preflight.StatusCode, string(preflight.RawBody))
	}
	if preflight.Headers["Access-Control-Allow-Origin"] != "" {
		t.Fatalf("expected no allow-origin header for unmatched preflight, got %q", preflight.Headers["Access-Control-Allow-Origin"])
	}
}

func TestQueueCORSPreflightRejectsMissingRequiredHeaders(t *testing.T) {
	svc := storage.New()
	preflight, err := svc.HandleRequest(storageCtx(t, http.MethodOptions, "https://acctest.queue.core.windows.net/work/messages", nil, map[string]string{
		"Origin":       "https://app.example",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("queue CORS preflight missing request method returned error: %v", err)
	}
	if preflight.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing-header queue CORS preflight status 400, got %d body=%s", preflight.StatusCode, string(preflight.RawBody))
	}
}

func TestQueueCORSActualGetAddsHeadersForMatchingStoredRule(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*,x-ms-approximate-messages-count</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work?comp=metadata", nil, map[string]string{
		"Origin":       "https://app.example",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue metadata with Origin returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get queue metadata status 200, got %d body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if getMetadata.Headers["Access-Control-Allow-Origin"] != "https://app.example" {
		t.Fatalf("expected allow-origin header for matching actual CORS request, got %q", getMetadata.Headers["Access-Control-Allow-Origin"])
	}
	if getMetadata.Headers["Access-Control-Expose-Headers"] != "x-ms-meta-data*,x-ms-approximate-messages-count" {
		t.Fatalf("expected expose-headers from matching CORS rule, got %q", getMetadata.Headers["Access-Control-Expose-Headers"])
	}
	if getMetadata.Headers["Vary"] != "Origin" {
		t.Fatalf("expected Vary Origin for exact-origin GET CORS match, got %q", getMetadata.Headers["Vary"])
	}
}

func TestQueueCORSActualGetSetsVaryForUnmatchedOrigin(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work?comp=metadata", nil, map[string]string{
		"Origin":       "https://other.example",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue metadata with unmatched Origin returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get queue metadata status 200, got %d body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if getMetadata.Headers["Access-Control-Allow-Origin"] != "" {
		t.Fatalf("expected no allow-origin header for unmatched actual CORS request, got %q", getMetadata.Headers["Access-Control-Allow-Origin"])
	}
	if getMetadata.Headers["Access-Control-Expose-Headers"] != "" {
		t.Fatalf("expected no expose-headers for unmatched actual CORS request, got %q", getMetadata.Headers["Access-Control-Expose-Headers"])
	}
	if getMetadata.Headers["Vary"] != "Origin" {
		t.Fatalf("expected Vary Origin for unmatched actual GET CORS request, got %q", getMetadata.Headers["Vary"])
	}
}

func TestQueueCORSActualGetRejectsDisallowedRequestHeaders(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://app.example</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work?comp=metadata", nil, map[string]string{
		"Origin":           "https://app.example",
		"x-ms-version":     "2023-11-03",
		"x-ms-meta-secret": "blocked",
	}))
	if err != nil {
		t.Fatalf("get queue metadata with disallowed CORS request header returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get queue metadata status 200, got %d body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if getMetadata.Headers["Access-Control-Allow-Origin"] != "" {
		t.Fatalf("expected no allow-origin header for disallowed actual CORS request header, got %q", getMetadata.Headers["Access-Control-Allow-Origin"])
	}
	if getMetadata.Headers["Access-Control-Expose-Headers"] != "" {
		t.Fatalf("expected no expose-headers for disallowed actual CORS request header, got %q", getMetadata.Headers["Access-Control-Expose-Headers"])
	}
	if getMetadata.Headers["Vary"] != "Origin" {
		t.Fatalf("expected Vary Origin for disallowed actual GET CORS request header, got %q", getMetadata.Headers["Vary"])
	}
}

func TestQueueCORSMatchesWildcardSubdomainOrigins(t *testing.T) {
	svc := storage.New()
	serviceURL := "https://acctest.queue.core.windows.net/?restype=service&comp=properties"
	properties := []byte(`<StorageServiceProperties><Cors><CorsRule><AllowedOrigins>https://*.contoso.com</AllowedOrigins><AllowedMethods>GET</AllowedMethods><MaxAgeInSeconds>200</MaxAgeInSeconds><ExposedHeaders>x-ms-meta-data*</ExposedHeaders><AllowedHeaders>x-ms-client-request-id</AllowedHeaders></CorsRule></Cors></StorageServiceProperties>`)
	setProperties, err := svc.HandleRequest(storageCtx(t, http.MethodPut, serviceURL, properties, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("set queue service CORS properties returned error: %v", err)
	}
	if setProperties.StatusCode != http.StatusAccepted {
		t.Fatalf("expected set queue service properties status 202, got %d body=%s", setProperties.StatusCode, string(setProperties.RawBody))
	}

	preflight, err := svc.HandleRequest(storageCtx(t, http.MethodOptions, "https://acctest.queue.core.windows.net/work/messages", nil, map[string]string{
		"Origin":                        "https://api.contoso.com",
		"Access-Control-Request-Method": "GET",
		"x-ms-version":                  "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("queue CORS preflight for wildcard subdomain returned error: %v", err)
	}
	if preflight.StatusCode != http.StatusOK {
		t.Fatalf("expected wildcard subdomain preflight status 200, got %d body=%s", preflight.StatusCode, string(preflight.RawBody))
	}
	if preflight.Headers["Access-Control-Allow-Origin"] != "https://api.contoso.com" {
		t.Fatalf("expected wildcard subdomain preflight allow-origin to echo request origin, got %q", preflight.Headers["Access-Control-Allow-Origin"])
	}

	createQueue, err := svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("create queue returned error: %v", err)
	}
	if createQueue.StatusCode != http.StatusCreated {
		t.Fatalf("expected create queue status 201, got %d body=%s", createQueue.StatusCode, string(createQueue.RawBody))
	}

	getMetadata, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work?comp=metadata", nil, map[string]string{
		"Origin":       "https://api.contoso.com",
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get queue metadata with wildcard subdomain Origin returned error: %v", err)
	}
	if getMetadata.StatusCode != http.StatusOK {
		t.Fatalf("expected get queue metadata status 200, got %d body=%s", getMetadata.StatusCode, string(getMetadata.RawBody))
	}
	if getMetadata.Headers["Access-Control-Allow-Origin"] != "https://api.contoso.com" {
		t.Fatalf("expected wildcard subdomain actual allow-origin to echo request origin, got %q", getMetadata.Headers["Access-Control-Allow-Origin"])
	}
	if getMetadata.Headers["Vary"] != "Origin" {
		t.Fatalf("expected Vary Origin for wildcard subdomain actual GET request, got %q", getMetadata.Headers["Vary"])
	}
}

func TestQueueClearMessagesDeletesAllMessages(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	for _, text := range []string{"msg1", "msg2"} {
		_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>`+text+`</MessageText></QueueMessage>`), map[string]string{
			"x-ms-version": "2023-11-03",
		}))
	}

	clearMessages, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("clear messages returned error: %v", err)
	}
	if clearMessages.StatusCode != http.StatusNoContent {
		t.Fatalf("expected clear messages status 204, got %d; body=%s", clearMessages.StatusCode, string(clearMessages.RawBody))
	}

	getMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?numofmessages=32", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("get messages after clear returned error: %v", err)
	}
	if strings.Contains(string(getMessages.RawBody), "<QueueMessage>") {
		t.Fatalf("expected clear messages to empty the queue, got: %s", string(getMessages.RawBody))
	}
}

func TestQueueUpdateMessageChangesTextAndPopReceipt(t *testing.T) {
	svc := storage.New()
	baseURL := "http://localhost:4577/devstoreaccount1-queue/work"
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, baseURL+"/messages", []byte(`<QueueMessage><MessageText>before</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	dequeue, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?visibilitytimeout=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("dequeue before update returned error: %v", err)
	}
	var messages struct {
		Messages []struct {
			MessageID  string `xml:"MessageId"`
			PopReceipt string `xml:"PopReceipt"`
			Text       string `xml:"MessageText"`
		} `xml:"QueueMessage"`
	}
	if err := xml.NewDecoder(bytes.NewReader(dequeue.RawBody)).Decode(&messages); err != nil && err != io.EOF {
		t.Fatalf("failed to decode dequeued message: %v", err)
	}
	if len(messages.Messages) != 1 {
		t.Fatalf("expected one dequeued message, got %d in %s", len(messages.Messages), string(dequeue.RawBody))
	}
	msg := messages.Messages[0]

	updateMessage, err := svc.HandleRequest(storageCtx(t, http.MethodPut, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt+"&visibilitytimeout=0", []byte(`<QueueMessage><MessageText>after</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("update message returned error: %v", err)
	}
	if updateMessage.StatusCode != http.StatusNoContent {
		t.Fatalf("expected update message status 204, got %d; body=%s", updateMessage.StatusCode, string(updateMessage.RawBody))
	}
	newReceipt := updateMessage.Headers["x-ms-popreceipt"]
	if newReceipt == "" || newReceipt == msg.PopReceipt || updateMessage.Headers["x-ms-time-next-visible"] == "" {
		t.Fatalf("expected updated message receipt and next-visible headers, got %v", updateMessage.Headers)
	}

	deleteWithOldReceipt, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+msg.PopReceipt, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete with old receipt returned error: %v", err)
	}
	if deleteWithOldReceipt.StatusCode != http.StatusNotFound {
		t.Fatalf("expected old pop receipt to be rejected, got %d", deleteWithOldReceipt.StatusCode)
	}

	peekMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, baseURL+"/messages?peekonly=true", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("peek after update returned error: %v", err)
	}
	peekBody := string(peekMessages.RawBody)
	if !strings.Contains(peekBody, "<MessageText>after</MessageText>") || strings.Contains(peekBody, "<MessageText>before</MessageText>") {
		t.Fatalf("expected updated message text in peek response, got: %s", peekBody)
	}

	deleteWithNewReceipt, err := svc.HandleRequest(storageCtx(t, http.MethodDelete, baseURL+"/messages/"+msg.MessageID+"?popreceipt="+newReceipt, nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("delete with new receipt returned error: %v", err)
	}
	if deleteWithNewReceipt.StatusCode != http.StatusNoContent {
		t.Fatalf("expected new pop receipt delete status 204, got %d", deleteWithNewReceipt.StatusCode)
	}
}

func TestQueuePeekMessagesDoesNotHideOrLockMessage(t *testing.T) {
	svc := storage.New()
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPut, "https://acctest.queue.core.windows.net/work", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	_, _ = svc.HandleRequest(storageCtx(t, http.MethodPost, "https://acctest.queue.core.windows.net/work/messages?visibilitytimeout=0", []byte(`<QueueMessage><MessageText>hello peek</MessageText></QueueMessage>`), map[string]string{
		"x-ms-version": "2023-11-03",
	}))

	peekMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work/messages?peekonly=true&numofmessages=1", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("peek messages returned error: %v", err)
	}
	peekBody := string(peekMessages.RawBody)
	if !strings.Contains(peekBody, "<MessageText>hello peek</MessageText>") {
		t.Fatalf("expected peek response to include visible message, got: %s", peekBody)
	}
	if strings.Contains(peekBody, "<PopReceipt>") {
		t.Fatalf("peek response must not include PopReceipt, got: %s", peekBody)
	}
	if strings.Contains(peekBody, "<TimeNextVisible>") {
		t.Fatalf("peek response must not include TimeNextVisible, got: %s", peekBody)
	}

	receiveMessages, err := svc.HandleRequest(storageCtx(t, http.MethodGet, "https://acctest.queue.core.windows.net/work/messages?numofmessages=1&visibilitytimeout=30", nil, map[string]string{
		"x-ms-version": "2023-11-03",
	}))
	if err != nil {
		t.Fatalf("receive messages returned error: %v", err)
	}
	var messages struct {
		Messages []struct {
			MessageID  string `xml:"MessageId"`
			PopReceipt string `xml:"PopReceipt"`
			Text       string `xml:"MessageText"`
		} `xml:"QueueMessage"`
	}
	if err := xml.NewDecoder(bytes.NewReader(receiveMessages.RawBody)).Decode(&messages); err != nil && err != io.EOF {
		t.Fatalf("failed to decode received messages: %v", err)
	}
	if len(messages.Messages) != 1 {
		t.Fatalf("expected receive after peek to return one message, got %d in %s", len(messages.Messages), string(receiveMessages.RawBody))
	}
	if messages.Messages[0].Text != "hello peek" {
		t.Fatalf("unexpected received message text: %q", messages.Messages[0].Text)
	}
	if messages.Messages[0].PopReceipt == "" {
		t.Fatalf("expected received message to include a pop receipt")
	}
}
