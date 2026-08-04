package redis

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestRedisLifecycleKeysAndRegenerateKey(t *testing.T) {
	svc := New()

	cacheURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis/cache-a?api-version=2024-11-01"
	cachePayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Basic","family":"C","capacity":1},
		"tags":{"env":"test"},
		"properties":{
			"enableNonSslPort":false,
			"minimumTlsVersion":"1.2",
			"redisConfiguration":{"maxmemory-policy":"allkeys-lru"}
		}
	}`)
	createCacheResp, err := svc.HandleRequest(redisCtx(t, http.MethodPut, cacheURL, cachePayload))
	if err != nil {
		t.Fatalf("create cache returned error: %v", err)
	}
	if createCacheResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create cache status 201, got %d; body=%s", createCacheResp.StatusCode, string(createCacheResp.RawBody))
	}
	cache := decodeRedisResponse(t, createCacheResp)
	if cache["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis/cache-a" {
		t.Fatalf("unexpected cache id: %v", cache["id"])
	}
	if cache["name"] != "cache-a" || cache["type"] != "Microsoft.Cache/Redis" || cache["location"] != "eastus" {
		t.Fatalf("unexpected cache identity fields: %v", cache)
	}
	cacheProps := cache["properties"].(map[string]any)
	if cacheProps["provisioningState"] != "Succeeded" || cacheProps["hostName"] != "cache-a.redis.cache.windows.net" || cacheProps["sslPort"].(float64) != 6380 {
		t.Fatalf("unexpected cache properties: %v", cacheProps)
	}

	listCachesResp, err := svc.HandleRequest(redisCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("list caches returned error: %v", err)
	}
	listedCaches := decodeRedisResponse(t, listCachesResp)
	if len(listedCaches["value"].([]any)) != 1 {
		t.Fatalf("expected one cache in list, got %v", listedCaches)
	}

	listKeysResp, err := svc.HandleRequest(redisCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis/cache-a/listKeys?api-version=2024-11-01", nil))
	if err != nil {
		t.Fatalf("list keys returned error: %v", err)
	}
	keys := decodeRedisResponse(t, listKeysResp)
	if keys["primaryKey"] != "cloudmock-cache-a-primary" || keys["secondaryKey"] != "cloudmock-cache-a-secondary" {
		t.Fatalf("unexpected initial keys: %v", keys)
	}

	regenerateResp, err := svc.HandleRequest(redisCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis/cache-a/regenerateKey?api-version=2024-11-01", []byte(`{"keyType":"Primary"}`)))
	if err != nil {
		t.Fatalf("regenerate key returned error: %v", err)
	}
	regenerated := decodeRedisResponse(t, regenerateResp)
	if regenerated["primaryKey"] != "cloudmock-cache-a-primary-r1" || regenerated["secondaryKey"] != "cloudmock-cache-a-secondary" {
		t.Fatalf("unexpected regenerated keys: %v", regenerated)
	}

	deleteCacheResp, err := svc.HandleRequest(redisCtx(t, http.MethodDelete, cacheURL, nil))
	if err != nil {
		t.Fatalf("delete cache returned error: %v", err)
	}
	if deleteCacheResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete cache status 202, got %d; body=%s", deleteCacheResp.StatusCode, string(deleteCacheResp.RawBody))
	}
}

func TestRedisTemplateProvisioning(t *testing.T) {
	svc := New()

	cacheResource := map[string]any{
		"type":     "Microsoft.Cache/Redis",
		"name":     "cache-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Basic", "family": "C", "capacity": 1},
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"minimumTlsVersion": "1.2",
		},
	}
	cacheResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", cacheResource)
	if err != nil {
		t.Fatalf("provision cache returned error: %v", err)
	}
	cache := cacheResult.(map[string]any)
	if cache["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Cache/Redis/cache-a" {
		t.Fatalf("unexpected provisioned cache id: %v", cache["id"])
	}
	if cache["type"] != "Microsoft.Cache/Redis" {
		t.Fatalf("unexpected provisioned cache type: %v", cache["type"])
	}
	props := cache["properties"].(map[string]any)
	if props["hostName"] != "cache-a.redis.cache.windows.net" {
		t.Fatalf("unexpected provisioned cache properties: %v", props)
	}
}

func redisCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	u, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{Method: method, URL: u, Host: u.Host}
	req.Header = http.Header{"Authorization": []string{"Bearer azure-token"}}
	return &service.RequestContext{
		Region:     "eastus",
		AccountID:  "sub-1",
		Action:     method,
		RawRequest: req,
		Body:       body,
	}
}

func decodeRedisResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
