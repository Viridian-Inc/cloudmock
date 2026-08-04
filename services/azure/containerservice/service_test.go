package containerservice

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestManagedClusterLifecycleCredentialsAndLists(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a?api-version=2026-02-01"
	createResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{
		"location":"eastus",
		"tags":{"env":"dev"},
		"properties":{
			"kubernetesVersion":"1.29",
			"dnsPrefix":"aks-a-dns",
			"agentPoolProfiles":[
				{"name":"nodepool1","count":2,"vmSize":"Standard_DS2_v2","osType":"Linux","mode":"System"}
			]
		}
	}`)))
	if err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeContainerServiceResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a" {
		t.Fatalf("unexpected cluster id: %v", created["id"])
	}
	if created["type"] != "Microsoft.ContainerService/managedClusters" || created["location"] != "eastus" {
		t.Fatalf("unexpected cluster identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" || properties["kubernetesVersion"] != "1.29" || properties["dnsPrefix"] != "aks-a-dns" {
		t.Fatalf("unexpected cluster properties: %v", properties)
	}
	pools := properties["agentPoolProfiles"].([]any)
	if len(pools) != 1 || pools[0].(map[string]any)["name"] != "nodepool1" || pools[0].(map[string]any)["count"].(float64) != 2 {
		t.Fatalf("unexpected agent pool profiles: %v", pools)
	}

	patchResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPatch, clusterURL, []byte(`{"tags":{"env":"prod"}}`)))
	if err != nil {
		t.Fatalf("patch managed cluster returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch status 200, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	patched := decodeContainerServiceResponse(t, patchResp)
	if patched["tags"].(map[string]any)["env"] != "prod" {
		t.Fatalf("expected patched tags, got %v", patched["tags"])
	}

	credentialURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-a/listClusterAdminCredential?api-version=2026-02-01"
	credentialResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, credentialURL, nil))
	if err != nil {
		t.Fatalf("list admin credential returned error: %v", err)
	}
	credentials := decodeContainerServiceResponse(t, credentialResp)
	kubeconfigs := credentials["kubeconfigs"].([]any)
	if len(kubeconfigs) != 1 || kubeconfigs[0].(map[string]any)["name"] != "clusterAdmin" {
		t.Fatalf("unexpected kubeconfigs: %v", kubeconfigs)
	}
	decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs[0].(map[string]any)["value"].(string))
	if err != nil {
		t.Fatalf("expected kubeconfig to be base64: %v", err)
	}
	if !strings.Contains(string(decodedKubeconfig), "aks-a") {
		t.Fatalf("expected kubeconfig to reference cluster, got %s", string(decodedKubeconfig))
	}

	otherRGURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.ContainerService/managedClusters/aks-b?api-version=2026-02-01"
	otherSubURL := "https://management.azure.com/subscriptions/sub-2/resourceGroups/rg-z/providers/Microsoft.ContainerService/managedClusters/aks-z?api-version=2026-02-01"
	for _, rawURL := range []string{otherRGURL, otherSubURL} {
		resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, rawURL, []byte(`{"location":"eastus","properties":{}}`)))
		if err != nil {
			t.Fatalf("create managed cluster %s returned error: %v", rawURL, err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("expected create status 201 for %s, got %d; body=%s", rawURL, resp.StatusCode, string(resp.RawBody))
		}
	}

	listRGResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters?api-version=2026-02-01", nil))
	if err != nil {
		t.Fatalf("list resource group clusters returned error: %v", err)
	}
	listRG := decodeContainerServiceResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "aks-a" {
		t.Fatalf("expected only rg-a cluster, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerService/managedClusters?api-version=2026-02-01", nil))
	if err != nil {
		t.Fatalf("list subscription clusters returned error: %v", err)
	}
	listSub := decodeContainerServiceResponse(t, listSubResp)
	subValues := listSub["value"].([]any)
	if len(subValues) != 2 || subValues[0].(map[string]any)["name"] != "aks-a" || subValues[1].(map[string]any)["name"] != "aks-b" {
		t.Fatalf("expected sub-1 clusters sorted by name, got %v", listSub)
	}

	deleteResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodDelete, clusterURL, nil))
	if err != nil {
		t.Fatalf("delete managed cluster returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
	getResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, clusterURL, nil))
	if err != nil {
		t.Fatalf("get deleted cluster returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted cluster status 404, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
}

func TestManagedClusterAgentPoolLifecycle(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool/agentPools?api-version=2026-04-01"
	listResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, listURL, nil))
	if err != nil {
		t.Fatalf("list agent pools returned error: %v", err)
	}
	listed := decodeContainerServiceResponse(t, listResp)
	if values := listed["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "nodepool1" {
		t.Fatalf("expected default nodepool, got %v", listed)
	}

	poolURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool/agentPools/userpool?api-version=2026-04-01"
	createPoolResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, poolURL, []byte(`{"properties":{"count":3,"vmSize":"Standard_D4s_v3","osType":"Linux","mode":"User"}}`)))
	if err != nil {
		t.Fatalf("create agent pool returned error: %v", err)
	}
	if createPoolResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create pool status 201, got %d; body=%s", createPoolResp.StatusCode, string(createPoolResp.RawBody))
	}
	createdPool := decodeContainerServiceResponse(t, createPoolResp)
	poolProps := createdPool["properties"].(map[string]any)
	if createdPool["type"] != "Microsoft.ContainerService/managedClusters/agentPools" || poolProps["count"].(float64) != 3 || poolProps["mode"] != "User" {
		t.Fatalf("unexpected created pool: %v", createdPool)
	}

	updatePoolResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, poolURL, []byte(`{"properties":{"count":4}}`)))
	if err != nil {
		t.Fatalf("update agent pool returned error: %v", err)
	}
	if updatePoolResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update pool status 200, got %d; body=%s", updatePoolResp.StatusCode, string(updatePoolResp.RawBody))
	}
	updatedPool := decodeContainerServiceResponse(t, updatePoolResp)
	if updatedPool["properties"].(map[string]any)["count"].(float64) != 4 {
		t.Fatalf("expected updated count, got %v", updatedPool)
	}

	deletePoolResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodDelete, poolURL, nil))
	if err != nil {
		t.Fatalf("delete agent pool returned error: %v", err)
	}
	if deletePoolResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete pool status 202, got %d; body=%s", deletePoolResp.StatusCode, string(deletePoolResp.RawBody))
	}
	getPoolResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, poolURL, nil))
	if err != nil {
		t.Fatalf("get deleted agent pool returned error: %v", err)
	}
	if getPoolResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted agent pool status 404, got %d; body=%s", getPoolResp.StatusCode, string(getPoolResp.RawBody))
	}
}

func TestAgentPoolUpgradeProfile(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool-up?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{
		"location":"eastus",
		"properties":{
			"kubernetesVersion":"1.29.7",
			"agentPoolProfiles":[
				{"name":"nodepool1","count":2,"vmSize":"Standard_DS2_v2","osType":"Linux","mode":"System"},
				{"name":"winpool","count":1,"vmSize":"Standard_D4s_v3","osType":"Windows","mode":"User"}
			]
		}
	}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool-up/agentPools/winpool/upgradeProfiles/default?api-version=2026-03-01"
	profileResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, profileURL, nil))
	if err != nil {
		t.Fatalf("get agent pool upgrade profile returned error: %v", err)
	}
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected agent pool upgrade profile status 200, got %d; body=%s", profileResp.StatusCode, string(profileResp.RawBody))
	}
	profile := decodeContainerServiceResponse(t, profileResp)
	if profile["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool-up/agentPools/winpool/upgradeprofiles/default" {
		t.Fatalf("unexpected agent pool upgrade profile id: %v", profile["id"])
	}
	if profile["name"] != "default" || profile["type"] != "Microsoft.ContainerService/managedClusters/agentPools/upgradeProfiles" {
		t.Fatalf("unexpected agent pool upgrade profile identity: %v", profile)
	}
	properties := profile["properties"].(map[string]any)
	if properties["kubernetesVersion"] != "1.29.7" || properties["osType"] != "Windows" {
		t.Fatalf("unexpected agent pool upgrade profile properties: %v", properties)
	}
	if properties["latestNodeImageVersion"] != "AKSWindows:2022:2026.03.01" {
		t.Fatalf("unexpected latest node image version: %v", properties)
	}
	assertUpgradeVersions(t, properties["upgrades"], []string{"1.30.0", "1.31.0"})

	missingPoolResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool-up/agentPools/missing/upgradeProfiles/default?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("get missing agent pool upgrade profile returned error: %v", err)
	}
	if missingPoolResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing pool status 404, got %d; body=%s", missingPoolResp.StatusCode, string(missingPoolResp.RawBody))
	}

	customProfileResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-pool-up/agentPools/winpool/upgradeProfiles/custom?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("get custom agent pool upgrade profile returned error: %v", err)
	}
	if customProfileResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected custom profile status 404, got %d; body=%s", customProfileResp.StatusCode, string(customProfileResp.RawBody))
	}
}

func TestAgentPoolUpgradeNodeImageVersion(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-node-image?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{
		"location":"eastus",
		"properties":{
			"agentPoolProfiles":[
				{"name":"nodepool1","count":2,"vmSize":"Standard_DS2_v2","osType":"Linux","mode":"System"}
			]
		}
	}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	upgradeURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-node-image/agentPools/nodepool1/upgradeNodeImageVersion?api-version=2026-03-01"
	upgradeResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, upgradeURL, nil))
	if err != nil {
		t.Fatalf("upgrade node image version returned error: %v", err)
	}
	if upgradeResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected upgrade node image status 202, got %d; body=%s", upgradeResp.StatusCode, string(upgradeResp.RawBody))
	}
	if upgradeResp.Headers["Azure-AsyncOperation"] == "" || upgradeResp.Headers["Location"] == "" || upgradeResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected node image upgrade operation headers, got %v", upgradeResp.Headers)
	}
	upgraded := decodeContainerServiceResponse(t, upgradeResp)
	props := upgraded["properties"].(map[string]any)
	if props["nodeImageVersion"] != "AKSUbuntu:2204:2026.03.01" || props["provisioningState"] != "UpgradingNodeImageVersion" {
		t.Fatalf("unexpected upgraded pool properties: %v", props)
	}

	poolURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-node-image/agentPools/nodepool1?api-version=2026-04-01"
	getResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, poolURL, nil))
	if err != nil {
		t.Fatalf("get upgraded agent pool returned error: %v", err)
	}
	gotPool := decodeContainerServiceResponse(t, getResp)
	gotProps := gotPool["properties"].(map[string]any)
	if gotProps["nodeImageVersion"] != "AKSUbuntu:2204:2026.03.01" || gotProps["provisioningState"] != "UpgradingNodeImageVersion" {
		t.Fatalf("expected stored upgraded pool state, got %v", gotProps)
	}

	clusterResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, clusterURL, nil))
	if err != nil {
		t.Fatalf("get cluster after node image upgrade returned error: %v", err)
	}
	cluster := decodeContainerServiceResponse(t, clusterResp)
	clusterProps := cluster["properties"].(map[string]any)
	clusterPools := clusterProps["agentPoolProfiles"].([]any)
	clusterPoolProps := clusterPools[0].(map[string]any)
	if clusterPoolProps["nodeImageVersion"] != "AKSUbuntu:2204:2026.03.01" || clusterPoolProps["provisioningState"] != "UpgradingNodeImageVersion" {
		t.Fatalf("expected cluster projection to include upgraded node image state, got %v", clusterPools)
	}

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-node-image/agentPools/missing/upgradeNodeImageVersion?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("upgrade missing pool returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing pool status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestAgentPoolAbortLatestOperation(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-abort?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{
		"location":"eastus",
		"properties":{
			"agentPoolProfiles":[
				{"name":"nodepool1","count":2,"vmSize":"Standard_DS2_v2","osType":"Linux","mode":"System"}
			]
		}
	}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	upgradeURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-abort/agentPools/nodepool1/upgradeNodeImageVersion?api-version=2026-03-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, upgradeURL, nil)); err != nil {
		t.Fatalf("upgrade node image version returned error: %v", err)
	} else if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected upgrade node image status 202, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	abortURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-abort/agentPools/nodepool1/abort?api-version=2026-03-01"
	abortResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, abortURL, nil))
	if err != nil {
		t.Fatalf("abort latest operation returned error: %v", err)
	}
	if abortResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected abort status 202, got %d; body=%s", abortResp.StatusCode, string(abortResp.RawBody))
	}
	if abortResp.Headers["Azure-AsyncOperation"] == "" || abortResp.Headers["Location"] == "" || abortResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected abort operation headers, got %v", abortResp.Headers)
	}

	poolURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-abort/agentPools/nodepool1?api-version=2026-04-01"
	getResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, poolURL, nil))
	if err != nil {
		t.Fatalf("get aborted agent pool returned error: %v", err)
	}
	gotPool := decodeContainerServiceResponse(t, getResp)
	if gotPool["properties"].(map[string]any)["provisioningState"] != "Canceled" {
		t.Fatalf("expected aborted pool to be canceled, got %v", gotPool)
	}

	idleResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, abortURL, nil))
	if err != nil {
		t.Fatalf("abort idle pool returned error: %v", err)
	}
	if idleResp.StatusCode != http.StatusConflict {
		t.Fatalf("expected idle abort status 409, got %d; body=%s", idleResp.StatusCode, string(idleResp.RawBody))
	}

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-abort/agentPools/missing/abort?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("abort missing pool returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing pool status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestManagedClusterUpgradeProfile(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-up?api-version=2026-02-01"
	createResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{
		"location":"eastus",
		"properties":{
			"kubernetesVersion":"1.29.7",
			"agentPoolProfiles":[
				{"name":"nodepool1","count":2,"vmSize":"Standard_DS2_v2","osType":"Linux","mode":"System"},
				{"name":"winpool","count":1,"vmSize":"Standard_D4s_v3","osType":"Windows","mode":"User"}
			]
		}
	}`)))
	if err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-up/upgradeProfiles/default?api-version=2026-03-01"
	profileResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, profileURL, nil))
	if err != nil {
		t.Fatalf("get upgrade profile returned error: %v", err)
	}
	if profileResp.StatusCode != http.StatusOK {
		t.Fatalf("expected upgrade profile status 200, got %d; body=%s", profileResp.StatusCode, string(profileResp.RawBody))
	}
	profile := decodeContainerServiceResponse(t, profileResp)
	if profile["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-up/upgradeprofiles/default" {
		t.Fatalf("unexpected upgrade profile id: %v", profile["id"])
	}
	if profile["name"] != "default" || profile["type"] != "Microsoft.ContainerService/managedClusters/upgradeprofiles" {
		t.Fatalf("unexpected upgrade profile identity: %v", profile)
	}

	properties := profile["properties"].(map[string]any)
	controlPlane := properties["controlPlaneProfile"].(map[string]any)
	if controlPlane["name"] != "master" || controlPlane["kubernetesVersion"] != "1.29.7" || controlPlane["osType"] != "Linux" {
		t.Fatalf("unexpected control plane profile: %v", controlPlane)
	}
	assertUpgradeVersions(t, controlPlane["upgrades"], []string{"1.30.0", "1.31.0"})

	pools := upgradePoolsByName(t, properties["agentPoolProfiles"])
	if len(pools) != 2 {
		t.Fatalf("expected two pool upgrade profiles, got %v", pools)
	}
	if pools["nodepool1"]["kubernetesVersion"] != "1.29.7" || pools["nodepool1"]["osType"] != "Linux" {
		t.Fatalf("unexpected nodepool upgrade profile: %v", pools["nodepool1"])
	}
	assertUpgradeVersions(t, pools["nodepool1"]["upgrades"], []string{"1.30.0", "1.31.0"})
	if pools["winpool"]["kubernetesVersion"] != "1.29.7" || pools["winpool"]["osType"] != "Windows" {
		t.Fatalf("unexpected winpool upgrade profile: %v", pools["winpool"])
	}

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/missing/upgradeProfiles/default?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("get missing upgrade profile returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing cluster status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}

	customResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-up/upgradeProfiles/custom?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("get custom upgrade profile returned error: %v", err)
	}
	if customResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected custom profile status 404, got %d; body=%s", customResp.StatusCode, string(customResp.RawBody))
	}
}

func TestManagedClusterStartStopActions(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-power?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{"kubernetesVersion":"1.29"}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	stopURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-power/stop?api-version=2026-03-01"
	stopResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, stopURL, nil))
	if err != nil {
		t.Fatalf("stop managed cluster returned error: %v", err)
	}
	if stopResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected stop status 202, got %d; body=%s", stopResp.StatusCode, string(stopResp.RawBody))
	}
	if stopResp.Headers["Location"] == "" || stopResp.Headers["Retry-After"] == "" {
		t.Fatalf("expected start/stop long-running-operation headers, got %v", stopResp.Headers)
	}

	getStoppedResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, clusterURL, nil))
	if err != nil {
		t.Fatalf("get stopped cluster returned error: %v", err)
	}
	assertPowerStateCode(t, decodeContainerServiceResponse(t, getStoppedResp), "Stopped")

	startURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-power/start?api-version=2026-03-01"
	startResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, startURL, nil))
	if err != nil {
		t.Fatalf("start managed cluster returned error: %v", err)
	}
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected start status 202, got %d; body=%s", startResp.StatusCode, string(startResp.RawBody))
	}

	getRunningResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, clusterURL, nil))
	if err != nil {
		t.Fatalf("get running cluster returned error: %v", err)
	}
	assertPowerStateCode(t, decodeContainerServiceResponse(t, getRunningResp), "Running")

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/missing/stop?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("stop missing cluster returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing cluster status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestManagedClusterRotationActions(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-rotate?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{"kubernetesVersion":"1.29"}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	for _, tt := range []struct {
		name string
		path string
	}{
		{name: "cluster certificates", path: "rotateClusterCertificates"},
		{name: "service account signing keys", path: "rotateServiceAccountSigningKeys"},
	} {
		rotateURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-rotate/" + tt.path + "?api-version=2026-03-01"
		resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, rotateURL, nil))
		if err != nil {
			t.Fatalf("rotate %s returned error: %v", tt.name, err)
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected rotate %s status 202, got %d; body=%s", tt.name, resp.StatusCode, string(resp.RawBody))
		}
		if resp.Headers["Location"] == "" || !strings.Contains(strings.ToLower(resp.Headers["Location"]), strings.ToLower(tt.path)) {
			t.Fatalf("expected rotate %s Location header to include action name, got %v", tt.name, resp.Headers)
		}
	}

	missingURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/missing/rotateClusterCertificates?api-version=2026-03-01"
	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, missingURL, nil))
	if err != nil {
		t.Fatalf("rotate missing cluster returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing cluster status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestManagedClusterMonitoringUserCredentials(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-monitor?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{"dnsPrefix":"aks-monitor-dns"}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	credentialURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-monitor/listClusterMonitoringUserCredential?api-version=2026-03-01&server-fqdn=private"
	credentialResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, credentialURL, nil))
	if err != nil {
		t.Fatalf("list monitoring user credential returned error: %v", err)
	}
	if credentialResp.StatusCode != http.StatusOK {
		t.Fatalf("expected monitoring credential status 200, got %d; body=%s", credentialResp.StatusCode, string(credentialResp.RawBody))
	}
	credentials := decodeContainerServiceResponse(t, credentialResp)
	kubeconfigs := credentials["kubeconfigs"].([]any)
	if len(kubeconfigs) != 1 || kubeconfigs[0].(map[string]any)["name"] != "clusterMonitoringUser" {
		t.Fatalf("unexpected monitoring kubeconfigs: %v", kubeconfigs)
	}
	decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs[0].(map[string]any)["value"].(string))
	if err != nil {
		t.Fatalf("expected kubeconfig to be base64: %v", err)
	}
	if !strings.Contains(string(decodedKubeconfig), "clusterMonitoringUser") || !strings.Contains(string(decodedKubeconfig), "private") {
		t.Fatalf("expected monitoring kubeconfig to reference credential and server fqdn, got %s", string(decodedKubeconfig))
	}

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/missing/listClusterMonitoringUserCredential?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("list missing monitoring credential returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing cluster status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestManagedClusterUserCredentialFormatQuery(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-user-format?api-version=2026-03-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{"dnsPrefix":"aks-user-format-dns"}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	for _, tt := range []struct {
		name         string
		format       string
		wantContains []string
		wantAbsent   []string
	}{
		{
			name:   "exec",
			format: "exec",
			wantContains: []string{
				"name: clusterUser_aks-user-format",
				"server: https://private.aks-user-format-dns.hcp.eastus.azmk8s.io",
				"exec:",
				"command: kubelogin",
				"- --login",
				"- azurecli",
				"- --server-id",
			},
			wantAbsent: []string{"token: cloudmock-aks-mock-token", "auth-provider:"},
		},
		{
			name:   "azure auth provider",
			format: "azure",
			wantContains: []string{
				"name: clusterUser_aks-user-format",
				"server: https://private.aks-user-format-dns.hcp.eastus.azmk8s.io",
				"auth-provider:",
				"name: azure",
				"apiserver-id:",
			},
			wantAbsent: []string{"token: cloudmock-aks-mock-token", "exec:"},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			credentialURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-user-format/listClusterUserCredential?api-version=2026-03-01&server-fqdn=private&format=" + tt.format
			credentialResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, credentialURL, nil))
			if err != nil {
				t.Fatalf("list user credential returned error: %v", err)
			}
			if credentialResp.StatusCode != http.StatusOK {
				t.Fatalf("expected user credential status 200, got %d; body=%s", credentialResp.StatusCode, string(credentialResp.RawBody))
			}
			credentials := decodeContainerServiceResponse(t, credentialResp)
			kubeconfigs := credentials["kubeconfigs"].([]any)
			if len(kubeconfigs) != 1 || kubeconfigs[0].(map[string]any)["name"] != "clusterUser" {
				t.Fatalf("unexpected user kubeconfigs: %v", kubeconfigs)
			}
			decodedKubeconfig, err := base64.StdEncoding.DecodeString(kubeconfigs[0].(map[string]any)["value"].(string))
			if err != nil {
				t.Fatalf("expected kubeconfig to be base64: %v", err)
			}
			kubeconfig := string(decodedKubeconfig)
			for _, want := range tt.wantContains {
				if !strings.Contains(kubeconfig, want) {
					t.Fatalf("expected %s kubeconfig to contain %q, got %s", tt.format, want, kubeconfig)
				}
			}
			for _, absent := range tt.wantAbsent {
				if strings.Contains(kubeconfig, absent) {
					t.Fatalf("expected %s kubeconfig not to contain %q, got %s", tt.format, absent, kubeconfig)
				}
			}
		})
	}
}

func TestManagedClusterRunCommandAndGetResult(t *testing.T) {
	svc := New()

	clusterURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-command?api-version=2026-02-01"
	if resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPut, clusterURL, []byte(`{"location":"eastus","properties":{"kubernetesVersion":"1.29"}}`))); err != nil {
		t.Fatalf("create managed cluster returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	runURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-command/runCommand?api-version=2026-03-01"
	runResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, runURL, []byte(`{"command":"kubectl get nodes","context":"","clusterToken":"fakeTokenPlaceholder"}`)))
	if err != nil {
		t.Fatalf("run command returned error: %v", err)
	}
	if runResp.StatusCode != http.StatusOK {
		t.Fatalf("expected run command status 200, got %d; body=%s", runResp.StatusCode, string(runResp.RawBody))
	}
	runResult := decodeContainerServiceResponse(t, runResp)
	commandID, ok := runResult["id"].(string)
	if !ok || commandID == "" {
		t.Fatalf("expected command id, got %v", runResult)
	}
	runProps := runResult["properties"].(map[string]any)
	if runProps["provisioningState"] != "succeeded" || runProps["exitCode"].(float64) != 0 {
		t.Fatalf("unexpected run command properties: %v", runProps)
	}
	if !strings.Contains(runProps["logs"].(string), "kubectl get nodes") {
		t.Fatalf("expected command logs to mention submitted command, got %v", runProps)
	}
	if runProps["startedAt"] == "" || runProps["finishedAt"] == "" {
		t.Fatalf("expected command timestamps, got %v", runProps)
	}

	resultURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-command/commandResults/" + commandID + "?api-version=2026-03-01"
	resultResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, resultURL, nil))
	if err != nil {
		t.Fatalf("get command result returned error: %v", err)
	}
	if resultResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get command result status 200, got %d; body=%s", resultResp.StatusCode, string(resultResp.RawBody))
	}
	result := decodeContainerServiceResponse(t, resultResp)
	if result["id"] != commandID {
		t.Fatalf("expected command result id %q, got %v", commandID, result)
	}
	resultProps := result["properties"].(map[string]any)
	if resultProps["logs"] != runProps["logs"] || resultProps["provisioningState"] != "succeeded" {
		t.Fatalf("unexpected command result properties: %v", resultProps)
	}

	missingResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerService/managedClusters/aks-command/commandResults/missing?api-version=2026-03-01", nil))
	if err != nil {
		t.Fatalf("get missing command result returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing command result status 404, got %d; body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}

	emptyResp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodPost, runURL, []byte(`{"command":""}`)))
	if err != nil {
		t.Fatalf("run empty command returned error: %v", err)
	}
	if emptyResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected empty command status 400, got %d; body=%s", emptyResp.StatusCode, string(emptyResp.RawBody))
	}
}

func TestListKubernetesVersions(t *testing.T) {
	svc := New()

	rawURL := "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerService/locations/eastus/kubernetesVersions?api-version=2026-03-01"
	resp, err := svc.HandleRequest(containerServiceCtx(t, http.MethodGet, rawURL, nil))
	if err != nil {
		t.Fatalf("list kubernetes versions returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected list kubernetes versions status 200, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeContainerServiceResponse(t, resp)
	values := body["values"].([]any)
	if len(values) < 3 {
		t.Fatalf("expected at least three kubernetes versions, got %v", body)
	}

	defaultVersion := values[0].(map[string]any)
	if defaultVersion["version"] != "1.29" || defaultVersion["isDefault"] != true {
		t.Fatalf("expected 1.29 to be default, got %v", defaultVersion)
	}
	capabilities := defaultVersion["capabilities"].(map[string]any)
	supportPlan := capabilities["supportPlan"].([]any)
	if len(supportPlan) != 1 || supportPlan[0] != "KubernetesOfficial" {
		t.Fatalf("unexpected support plan: %v", capabilities)
	}
	patchVersions := defaultVersion["patchVersions"].(map[string]any)
	patch := patchVersions["1.29.7"].(map[string]any)
	upgrades := patch["upgrades"].([]any)
	if len(upgrades) != 2 || upgrades[0] != "1.30.0" || upgrades[1] != "1.31.0" {
		t.Fatalf("unexpected patch upgrades: %v", patchVersions)
	}

	latestVersion := values[len(values)-1].(map[string]any)
	if latestVersion["version"] != "1.31" || latestVersion["isPreview"] != true {
		t.Fatalf("expected latest version to be preview, got %v", latestVersion)
	}
}

func TestServiceKeysIncludeAKSAPIVersions(t *testing.T) {
	svc := New()
	keys := svc.ServiceKeys()
	expected := []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerService/managedClusters", APIVersion: "2026-02-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerService/managedClusters", APIVersion: "2026-03-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerService/managedClusters", APIVersion: "2026-04-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerService/managedClusters", APIVersion: "2024-04-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerService/locations", APIVersion: "2026-03-01"},
	}
	for _, want := range expected {
		found := false
		for _, got := range keys {
			if got == want {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected service key %#v in %#v", want, keys)
		}
	}
}

func assertPowerStateCode(t *testing.T, cluster map[string]any, expected string) {
	t.Helper()
	properties, ok := cluster["properties"].(map[string]any)
	if !ok {
		t.Fatalf("expected cluster properties in %v", cluster)
	}
	powerState, ok := properties["powerState"].(map[string]any)
	if !ok {
		t.Fatalf("expected powerState in properties %v", properties)
	}
	if got := powerState["code"]; got != expected {
		t.Fatalf("expected powerState code %q, got %v in %v", expected, got, powerState)
	}
}

func assertUpgradeVersions(t *testing.T, raw any, expected []string) {
	t.Helper()
	upgrades := raw.([]any)
	if len(upgrades) != len(expected) {
		t.Fatalf("expected upgrades %v, got %v", expected, upgrades)
	}
	for i, want := range expected {
		got := upgrades[i].(map[string]any)["kubernetesVersion"]
		if got != want {
			t.Fatalf("expected upgrade %d to be %q, got %v in %v", i, want, got, upgrades)
		}
	}
}

func upgradePoolsByName(t *testing.T, raw any) map[string]map[string]any {
	t.Helper()
	values := raw.([]any)
	out := make(map[string]map[string]any, len(values))
	for _, value := range values {
		pool := value.(map[string]any)
		out[pool["name"].(string)] = pool
	}
	return out
}

func containerServiceCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeContainerServiceResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
