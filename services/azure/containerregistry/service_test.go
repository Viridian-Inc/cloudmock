package containerregistry

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestRegistryLifecycleAndCredentials(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra?api-version=2025-11-01"
	payload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard"},
		"tags":{"env":"test"},
		"properties":{
			"adminUserEnabled":true,
			"publicNetworkAccess":"Enabled"
		}
	}`)

	createResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, payload))
	if err != nil {
		t.Fatalf("create registry returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeContainerRegistryResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra" {
		t.Fatalf("unexpected registry id: %v", created["id"])
	}
	if created["name"] != "acra" || created["type"] != "Microsoft.ContainerRegistry/registries" || created["location"] != "eastus" {
		t.Fatalf("unexpected registry identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" || properties["loginServer"] != "acra.azurecr.io" {
		t.Fatalf("unexpected registry properties: %v", properties)
	}

	getResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, registryURL, nil))
	if err != nil {
		t.Fatalf("get registry returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get registry status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("list registries returned error: %v", err)
	}
	listed := decodeContainerRegistryResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one registry in list, got %d in %v", len(values), listed)
	}

	listCredentialsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/listCredentials?api-version=2025-11-01"
	listCredentialsResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, listCredentialsURL, nil))
	if err != nil {
		t.Fatalf("list credentials returned error: %v", err)
	}
	if listCredentialsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list credentials status 200, got %d; body=%s", listCredentialsResp.StatusCode, string(listCredentialsResp.RawBody))
	}
	credentials := decodeContainerRegistryResponse(t, listCredentialsResp)
	if credentials["username"] != "acra" {
		t.Fatalf("unexpected credentials username: %v", credentials["username"])
	}
	passwords := credentials["passwords"].([]any)
	if len(passwords) != 2 {
		t.Fatalf("expected two credentials, got %d in %v", len(passwords), credentials)
	}
	firstPassword := passwords[0].(map[string]any)["value"].(string)

	regenerateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/regenerateCredential?api-version=2025-11-01"
	regenerateResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, regenerateURL, []byte(`{"name":"password"}`)))
	if err != nil {
		t.Fatalf("regenerate credential returned error: %v", err)
	}
	if regenerateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected regenerate credential status 200, got %d; body=%s", regenerateResp.StatusCode, string(regenerateResp.RawBody))
	}
	regenerated := decodeContainerRegistryResponse(t, regenerateResp)
	regeneratedPassword := regenerated["passwords"].([]any)[0].(map[string]any)["value"].(string)
	if regeneratedPassword == firstPassword {
		t.Fatalf("expected regenerated password to change from %q", firstPassword)
	}

	deleteResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodDelete, registryURL, nil))
	if err != nil {
		t.Fatalf("delete registry returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete registry status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
}

func TestRegistryTemplateProvisioning(t *testing.T) {
	svc := New()

	result, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.ContainerRegistry/registries",
		"name":     "acra",
		"location": "eastus",
		"sku":      map[string]any{"name": "Premium"},
		"tags":     map[string]any{"env": "test"},
		"properties": map[string]any{
			"adminUserEnabled": true,
		},
	})
	if err != nil {
		t.Fatalf("provision registry returned error: %v", err)
	}
	registry := result.(map[string]any)
	if registry["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra" {
		t.Fatalf("unexpected provisioned registry id: %v", registry["id"])
	}
	if registry["type"] != "Microsoft.ContainerRegistry/registries" {
		t.Fatalf("unexpected provisioned registry type: %v", registry["type"])
	}
	properties := registry["properties"].(map[string]any)
	if properties["loginServer"] != "acra.azurecr.io" {
		t.Fatalf("unexpected provisioned registry login server: %v", properties["loginServer"])
	}
}

func TestCheckNameAvailabilityAndReplications(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrtaken?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`))); err != nil {
		t.Fatalf("create registry returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	checkURL := "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerRegistry/checkNameAvailability?api-version=2025-11-01"
	takenResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, checkURL, []byte(`{"name":"acrtaken","type":"Microsoft.ContainerRegistry/registries"}`)))
	if err != nil {
		t.Fatalf("check taken name returned error: %v", err)
	}
	if takenResp.StatusCode != http.StatusOK {
		t.Fatalf("expected check taken status 200, got %d; body=%s", takenResp.StatusCode, string(takenResp.RawBody))
	}
	taken := decodeContainerRegistryResponse(t, takenResp)
	if taken["nameAvailable"] != false || taken["reason"] != "AlreadyExists" {
		t.Fatalf("expected taken name response, got %v", taken)
	}

	freeResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, checkURL, []byte(`{"name":"acrfreexyz","type":"Microsoft.ContainerRegistry/registries"}`)))
	if err != nil {
		t.Fatalf("check free name returned error: %v", err)
	}
	free := decodeContainerRegistryResponse(t, freeResp)
	if free["nameAvailable"] != true {
		t.Fatalf("expected free name response, got %v", free)
	}

	invalidResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, checkURL, []byte(`{"name":"bad-name","type":"Microsoft.ContainerRegistry/registries"}`)))
	if err != nil {
		t.Fatalf("check invalid name returned error: %v", err)
	}
	invalid := decodeContainerRegistryResponse(t, invalidResp)
	if invalid["nameAvailable"] != false || invalid["reason"] != "Invalid" {
		t.Fatalf("expected invalid name response, got %v", invalid)
	}

	replicationsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrtaken/replications?api-version=2025-11-01"
	replicationsResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, replicationsURL, nil))
	if err != nil {
		t.Fatalf("list replications returned error: %v", err)
	}
	if replicationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected list replications status 200, got %d; body=%s", replicationsResp.StatusCode, string(replicationsResp.RawBody))
	}
	replications := decodeContainerRegistryResponse(t, replicationsResp)
	if values := replications["value"].([]any); len(values) != 0 {
		t.Fatalf("expected empty replications list, got %v", replications)
	}
}

func TestRegistryPatchUpdatesMutableFields(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrpatch?api-version=2025-11-01"
	createResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{
		"location":"eastus",
		"sku":{"name":"Basic","tier":"Basic"},
		"tags":{"env":"dev","keep":"true"},
		"properties":{
			"adminUserEnabled":false,
			"publicNetworkAccess":"Enabled"
		}
	}`)))
	if err != nil {
		t.Fatalf("create registry returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	patchResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPatch, registryURL, []byte(`{
		"tags":{"env":"prod"},
		"sku":{"name":"Standard"},
		"properties":{
			"adminUserEnabled":true,
			"publicNetworkAccess":"Disabled",
			"roleAssignmentMode":"AbacRepositoryPermissions"
		}
	}`)))
	if err != nil {
		t.Fatalf("patch registry returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch status 200, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	patched := decodeContainerRegistryResponse(t, patchResp)
	if tags := patched["tags"].(map[string]any); len(tags) != 1 || tags["env"] != "prod" {
		t.Fatalf("expected patch to replace tags, got %v", tags)
	}
	sku := patched["sku"].(map[string]any)
	if sku["name"] != "Standard" || sku["tier"] != "Standard" {
		t.Fatalf("expected Standard sku and tier, got %v", sku)
	}
	properties := patched["properties"].(map[string]any)
	if properties["adminUserEnabled"] != true || properties["publicNetworkAccess"] != "Disabled" || properties["roleAssignmentMode"] != "AbacRepositoryPermissions" {
		t.Fatalf("expected mutable properties to update, got %v", properties)
	}
	if properties["loginServer"] != "acrpatch.azurecr.io" || properties["provisioningState"] != "Succeeded" {
		t.Fatalf("expected immutable/default properties preserved, got %v", properties)
	}

	getResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, registryURL, nil))
	if err != nil {
		t.Fatalf("get patched registry returned error: %v", err)
	}
	got := decodeContainerRegistryResponse(t, getResp)
	gotProperties := got["properties"].(map[string]any)
	if gotProperties["adminUserEnabled"] != true || got["sku"].(map[string]any)["name"] != "Standard" {
		t.Fatalf("expected patch to persist, got %v", got)
	}
}

func TestListSubscriptionRegistriesAndUsages(t *testing.T) {
	svc := New()

	createURLs := []string{
		"https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.ContainerRegistry/registries/acrb?api-version=2025-11-01",
		"https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra?api-version=2025-11-01",
		"https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.ContainerRegistry/registries/acrz?api-version=2025-11-01",
	}
	for _, createURL := range createURLs {
		resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, createURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`)))
		if err != nil {
			t.Fatalf("create registry %s returned error: %v", createURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create status 201 for %s, got %d; body=%s", createURL, resp.StatusCode, string(resp.RawBody))
		}
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerRegistry/registries?api-version=2025-11-01"
	listResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list subscription registries returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription list status 200, got %d; body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	listed := decodeContainerRegistryResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected two sub-1 registries, got %d in %v", len(values), listed)
	}
	if values[0].(map[string]any)["name"] != "acra" || values[1].(map[string]any)["name"] != "acrb" {
		t.Fatalf("expected subscription list sorted by name, got %v", values)
	}

	usagesURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra/listUsages?api-version=2025-11-01"
	usagesResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, usagesURL, nil))
	if err != nil {
		t.Fatalf("list usages returned error: %v", err)
	}
	if usagesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected usages status 200, got %d; body=%s", usagesResp.StatusCode, string(usagesResp.RawBody))
	}
	usages := decodeContainerRegistryResponse(t, usagesResp)
	usageValues := usages["value"].([]any)
	if len(usageValues) != 1 {
		t.Fatalf("expected one static usage, got %v", usages)
	}
	sizeUsage := usageValues[0].(map[string]any)
	if sizeUsage["name"] != "Size" || sizeUsage["unit"] != "Bytes" || sizeUsage["currentValue"].(float64) != 0 {
		t.Fatalf("unexpected size usage: %v", sizeUsage)
	}
	if sizeUsage["limit"].(float64) <= 0 {
		t.Fatalf("expected positive size limit, got %v", sizeUsage)
	}
}

func TestImportImageAcceptedNoop(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrimport?api-version=2025-11-01"
	createResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`)))
	if err != nil {
		t.Fatalf("create registry returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	importURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrimport/importImage?api-version=2025-11-01"
	importResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, importURL, []byte(`{
		"source":{"sourceImage":"library/alpine:latest"},
		"targetTags":["imports/alpine:v1"],
		"mode":"Force"
	}`)))
	if err != nil {
		t.Fatalf("import image returned error: %v", err)
	}
	if importResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected import image status 202, got %d; body=%s", importResp.StatusCode, string(importResp.RawBody))
	}
	if len(importResp.RawBody) != 0 {
		t.Fatalf("expected import image no-op to return an empty body, got %s", string(importResp.RawBody))
	}
}

func TestRegistryDataPlaneManifestTagsAndCatalog(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acra?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`))); err != nil {
		t.Fatalf("create registry returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	pingResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/", nil))
	if err != nil {
		t.Fatalf("registry ping returned error: %v", err)
	}
	if pingResp.StatusCode != http.StatusOK {
		t.Fatalf("expected registry ping status 200, got %d; body=%s", pingResp.StatusCode, string(pingResp.RawBody))
	}
	if pingResp.Headers["Docker-Distribution-API-Version"] != "registry/2.0" {
		t.Fatalf("expected registry API header, got %v", pingResp.Headers)
	}

	catalogResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/_catalog", nil))
	if err != nil {
		t.Fatalf("empty catalog returned error: %v", err)
	}
	emptyCatalog := decodeContainerRegistryResponse(t, catalogResp)
	if repositories := emptyCatalog["repositories"].([]any); len(repositories) != 0 {
		t.Fatalf("expected empty catalog, got %v", emptyCatalog)
	}

	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.docker.distribution.manifest.v2+json","config":{"mediaType":"application/vnd.docker.container.image.v1+json","digest":"sha256:cfg","size":2},"layers":[]}`)
	putCtx := containerRegistryCtx(t, http.MethodPut, "https://acra.azurecr.io/v2/library/alpine/manifests/latest", manifest)
	putCtx.RawRequest.Header.Set("Content-Type", "application/vnd.docker.distribution.manifest.v2+json")
	putResp, err := svc.HandleRequest(putCtx)
	if err != nil {
		t.Fatalf("put manifest returned error: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected put manifest status 201, got %d; body=%s", putResp.StatusCode, string(putResp.RawBody))
	}
	sum := sha256.Sum256(manifest)
	expectedDigest := "sha256:" + hex.EncodeToString(sum[:])
	if putResp.Headers["Docker-Content-Digest"] != expectedDigest {
		t.Fatalf("expected digest %q, got headers %v", expectedDigest, putResp.Headers)
	}
	if putResp.Headers["Location"] != "/v2/library/alpine/manifests/latest" {
		t.Fatalf("unexpected manifest location header: %v", putResp.Headers)
	}

	getResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/library/alpine/manifests/latest", nil))
	if err != nil {
		t.Fatalf("get manifest by tag returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get manifest status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	if string(getResp.RawBody) != string(manifest) {
		t.Fatalf("expected manifest body %s, got %s", string(manifest), string(getResp.RawBody))
	}
	if getResp.RawContentType != "application/vnd.docker.distribution.manifest.v2+json" {
		t.Fatalf("unexpected manifest content type %q", getResp.RawContentType)
	}
	if getResp.Headers["Docker-Content-Digest"] != expectedDigest {
		t.Fatalf("expected manifest digest header %q, got %v", expectedDigest, getResp.Headers)
	}

	headResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodHead, "https://acra.azurecr.io/v2/library/alpine/manifests/latest", nil))
	if err != nil {
		t.Fatalf("head manifest returned error: %v", err)
	}
	if headResp.StatusCode != http.StatusOK || len(headResp.RawBody) != 0 {
		t.Fatalf("expected empty 200 HEAD manifest response, got status=%d body=%s", headResp.StatusCode, string(headResp.RawBody))
	}
	if headResp.Headers["Docker-Content-Digest"] != expectedDigest {
		t.Fatalf("expected HEAD digest header %q, got %v", expectedDigest, headResp.Headers)
	}

	digestResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/library/alpine/manifests/"+expectedDigest, nil))
	if err != nil {
		t.Fatalf("get manifest by digest returned error: %v", err)
	}
	if digestResp.StatusCode != http.StatusOK || string(digestResp.RawBody) != string(manifest) {
		t.Fatalf("expected digest lookup to return manifest, got status=%d body=%s", digestResp.StatusCode, string(digestResp.RawBody))
	}

	tagsResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/library/alpine/tags/list", nil))
	if err != nil {
		t.Fatalf("list tags returned error: %v", err)
	}
	tags := decodeContainerRegistryResponse(t, tagsResp)
	if tags["name"] != "library/alpine" {
		t.Fatalf("unexpected tags repository name: %v", tags)
	}
	tagValues := tags["tags"].([]any)
	if len(tagValues) != 1 || tagValues[0] != "latest" {
		t.Fatalf("expected latest tag, got %v", tags)
	}

	catalogResp, err = svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/_catalog", nil))
	if err != nil {
		t.Fatalf("catalog returned error: %v", err)
	}
	catalog := decodeContainerRegistryResponse(t, catalogResp)
	repositories := catalog["repositories"].([]any)
	if len(repositories) != 1 || repositories[0] != "library/alpine" {
		t.Fatalf("expected pushed repository in catalog, got %v", catalog)
	}

	deleteResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodDelete, "https://acra.azurecr.io/v2/library/alpine/manifests/latest", nil))
	if err != nil {
		t.Fatalf("delete manifest returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete manifest status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	deletedResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acra.azurecr.io/v2/library/alpine/manifests/latest", nil))
	if err != nil {
		t.Fatalf("get deleted manifest returned error: %v", err)
	}
	if deletedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted manifest lookup status 404, got %d; body=%s", deletedResp.StatusCode, string(deletedResp.RawBody))
	}
}

func TestRegistryDataPlaneLocalRouteAndMissingResources(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrlocal?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`))); err != nil {
		t.Fatalf("create registry returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	manifest := []byte(`{"schemaVersion":2,"mediaType":"application/vnd.oci.image.manifest.v1+json","config":{"mediaType":"application/vnd.oci.empty.v1+json","digest":"sha256:cfg","size":2},"layers":[]}`)
	putCtx := containerRegistryCtx(t, http.MethodPut, "http://localhost:4577/acrlocal-acr/v2/team/app/manifests/v1", manifest)
	putCtx.RawRequest.Header.Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
	putResp, err := svc.HandleRequest(putCtx)
	if err != nil {
		t.Fatalf("local put manifest returned error: %v", err)
	}
	if putResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected local put manifest status 201, got %d; body=%s", putResp.StatusCode, string(putResp.RawBody))
	}

	getResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "http://localhost:4577/acrlocal-acr/v2/team/app/manifests/v1", nil))
	if err != nil {
		t.Fatalf("local get manifest returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK || string(getResp.RawBody) != string(manifest) {
		t.Fatalf("expected local get manifest to return body, got status=%d body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	missingRegistryResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://missing.azurecr.io/v2/", nil))
	if err != nil {
		t.Fatalf("missing registry ping returned error: %v", err)
	}
	if missingRegistryResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing registry status 404, got %d; body=%s", missingRegistryResp.StatusCode, string(missingRegistryResp.RawBody))
	}

	missingManifestResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "http://localhost:4577/acrlocal-acr/v2/team/app/manifests/missing", nil))
	if err != nil {
		t.Fatalf("missing manifest returned error: %v", err)
	}
	if missingManifestResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing manifest status 404, got %d; body=%s", missingManifestResp.StatusCode, string(missingManifestResp.RawBody))
	}
}

func TestRegistryDataPlaneBlobUploadReadAndDelete(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrblob?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`))); err != nil {
		t.Fatalf("create registry returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	startResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, "https://acrblob.azurecr.io/v2/library/alpine/blobs/uploads/", nil))
	if err != nil {
		t.Fatalf("start upload returned error: %v", err)
	}
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected start upload status 202, got %d; body=%s", startResp.StatusCode, string(startResp.RawBody))
	}
	uploadLocation := startResp.Headers["Location"]
	uploadID := startResp.Headers["Docker-Upload-UUID"]
	if uploadLocation == "" || uploadID == "" {
		t.Fatalf("expected upload location and uuid headers, got %v", startResp.Headers)
	}
	if startResp.Headers["Range"] != "bytes=0-0" || startResp.Headers["Content-Length"] != "0" {
		t.Fatalf("unexpected start upload headers: %v", startResp.Headers)
	}

	chunk := []byte("hello-layer")
	patchResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPatch, "https://acrblob.azurecr.io"+uploadLocation, chunk))
	if err != nil {
		t.Fatalf("upload chunk returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected upload chunk status 202, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	if patchResp.Headers["Docker-Upload-UUID"] != uploadID || patchResp.Headers["Location"] != uploadLocation {
		t.Fatalf("expected upload headers to be preserved, got %v", patchResp.Headers)
	}
	if patchResp.Headers["Range"] != "bytes=0-10" {
		t.Fatalf("expected uploaded range bytes=0-10, got %v", patchResp.Headers)
	}

	sum := sha256.Sum256(chunk)
	digest := "sha256:" + hex.EncodeToString(sum[:])
	completeResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, "https://acrblob.azurecr.io"+uploadLocation+"?digest="+url.QueryEscape(digest), nil))
	if err != nil {
		t.Fatalf("complete upload returned error: %v", err)
	}
	if completeResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected complete upload status 201, got %d; body=%s", completeResp.StatusCode, string(completeResp.RawBody))
	}
	if completeResp.Headers["Docker-Content-Digest"] != digest {
		t.Fatalf("expected digest header %q, got %v", digest, completeResp.Headers)
	}
	if completeResp.Headers["Location"] != "/v2/library/alpine/blobs/"+digest {
		t.Fatalf("unexpected complete upload location: %v", completeResp.Headers)
	}

	headResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodHead, "https://acrblob.azurecr.io/v2/library/alpine/blobs/"+digest, nil))
	if err != nil {
		t.Fatalf("head blob returned error: %v", err)
	}
	if headResp.StatusCode != http.StatusOK || len(headResp.RawBody) != 0 {
		t.Fatalf("expected empty 200 HEAD blob response, got status=%d body=%s", headResp.StatusCode, string(headResp.RawBody))
	}
	if headResp.Headers["Docker-Content-Digest"] != digest || headResp.Headers["Content-Length"] != "11" {
		t.Fatalf("unexpected HEAD blob headers: %v", headResp.Headers)
	}

	getResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acrblob.azurecr.io/v2/library/alpine/blobs/"+digest, nil))
	if err != nil {
		t.Fatalf("get blob returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK || string(getResp.RawBody) != string(chunk) {
		t.Fatalf("expected blob body, got status=%d body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	if getResp.Headers["Docker-Content-Digest"] != digest || getResp.Headers["Content-Length"] != "11" {
		t.Fatalf("unexpected GET blob headers: %v", getResp.Headers)
	}

	deleteResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodDelete, "https://acrblob.azurecr.io/v2/library/alpine/blobs/"+digest, nil))
	if err != nil {
		t.Fatalf("delete blob returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete blob status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	missingResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodGet, "https://acrblob.azurecr.io/v2/library/alpine/blobs/"+digest, nil))
	if err != nil {
		t.Fatalf("get deleted blob returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted blob status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestRegistryDataPlaneBlobUploadDigestValidationAndCancel(t *testing.T) {
	svc := New()

	registryURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerRegistry/registries/acrbloberr?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, registryURL, []byte(`{"location":"eastus","sku":{"name":"Basic"}}`))); err != nil {
		t.Fatalf("create registry returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create registry status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	startResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPost, "https://acrbloberr.azurecr.io/v2/team/app/blobs/uploads/", nil))
	if err != nil {
		t.Fatalf("start upload returned error: %v", err)
	}
	uploadLocation := startResp.Headers["Location"]
	if uploadLocation == "" {
		t.Fatalf("expected upload location, got %v", startResp.Headers)
	}
	if _, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPatch, "https://acrbloberr.azurecr.io"+uploadLocation, []byte("wrong-digest"))); err != nil {
		t.Fatalf("upload chunk returned error: %v", err)
	}

	badDigest := "sha256:" + strings.Repeat("0", 64)
	badComplete, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, "https://acrbloberr.azurecr.io"+uploadLocation+"?digest="+badDigest, nil))
	if err != nil {
		t.Fatalf("bad complete returned error: %v", err)
	}
	if badComplete.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected bad digest status 400, got %d; body=%s", badComplete.StatusCode, string(badComplete.RawBody))
	}
	badBody := decodeContainerRegistryResponse(t, badComplete)
	errors := badBody["errors"].([]any)
	if len(errors) != 1 || errors[0].(map[string]any)["code"] != "DIGEST_INVALID" {
		t.Fatalf("expected Docker registry DIGEST_INVALID error, got %v", badBody)
	}

	missingBlobResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodHead, "https://acrbloberr.azurecr.io/v2/team/app/blobs/"+badDigest, nil))
	if err != nil {
		t.Fatalf("missing blob HEAD returned error: %v", err)
	}
	if missingBlobResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing blob status 404, got %d; body=%s", missingBlobResp.StatusCode, string(missingBlobResp.RawBody))
	}

	cancelResp, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodDelete, "https://acrbloberr.azurecr.io"+uploadLocation, nil))
	if err != nil {
		t.Fatalf("cancel upload returned error: %v", err)
	}
	if cancelResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected cancel upload status 204, got %d; body=%s", cancelResp.StatusCode, string(cancelResp.RawBody))
	}
	canceledComplete, err := svc.HandleRequest(containerRegistryCtx(t, http.MethodPut, "https://acrbloberr.azurecr.io"+uploadLocation+"?digest="+badDigest, nil))
	if err != nil {
		t.Fatalf("complete canceled upload returned error: %v", err)
	}
	if canceledComplete.StatusCode != http.StatusNotFound {
		t.Fatalf("expected canceled upload status 404, got %d; body=%s", canceledComplete.StatusCode, string(canceledComplete.RawBody))
	}
}

func containerRegistryCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeContainerRegistryResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
