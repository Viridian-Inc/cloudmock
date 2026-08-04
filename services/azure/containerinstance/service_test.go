package containerinstance

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/routing"
	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestContainerGroupLifecycleAndLists(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"containers":[{
				"name":"web",
				"properties":{
					"image":"nginx:1.27",
					"resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}},
					"ports":[{"port":80,"protocol":"TCP"}],
					"environmentVariables":[{"name":"MODE","value":"test"}]
				}
			}],
			"imageRegistryCredentials":[{"server":"index.docker.io","username":"cloudmock","password":"secret"}],
			"ipAddress":{"type":"Public","ports":[{"port":80,"protocol":"TCP"}],"dnsNameLabel":"cg-a"},
			"osType":"Linux",
			"restartPolicy":"Never",
			"volumes":[{"name":"scratch","emptyDir":{}}]
		}
	}`)

	createResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload))
	if err != nil {
		t.Fatalf("create container group returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create container group status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeContainerInstanceResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a" {
		t.Fatalf("unexpected container group id: %v", created["id"])
	}
	if created["name"] != "cg-a" || created["type"] != "Microsoft.ContainerInstance/containerGroups" || created["location"] != "eastus" {
		t.Fatalf("unexpected container group identity fields: %v", created)
	}
	properties := created["properties"].(map[string]any)
	if properties["provisioningState"] != "Succeeded" || properties["instanceView"].(map[string]any)["state"] != "Running" {
		t.Fatalf("unexpected container group state: %v", properties)
	}
	if properties["restartPolicy"] != "Never" || properties["osType"] != "Linux" {
		t.Fatalf("expected request properties to be preserved, got %v", properties)
	}
	containers := properties["containers"].([]any)
	if len(containers) != 1 || containers[0].(map[string]any)["name"] != "web" {
		t.Fatalf("expected web container to be preserved, got %v", containers)
	}
	containerProps := containers[0].(map[string]any)["properties"].(map[string]any)
	if containerProps["image"] != "nginx:1.27" {
		t.Fatalf("expected container image to be preserved, got %v", containerProps)
	}
	if containerProps["instanceView"].(map[string]any)["currentState"].(map[string]any)["state"] != "Running" {
		t.Fatalf("expected deterministic container instance view, got %v", containerProps["instanceView"])
	}

	updateResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, []byte(`{
		"location":"eastus2",
		"tags":{"env":"prod"},
		"properties":{
			"containers":[{"name":"worker","properties":{"image":"alpine:latest","resources":{"requests":{"cpu":0.5,"memoryInGB":1.0}}}}],
			"osType":"Linux",
			"restartPolicy":"OnFailure"
		}
	}`)))
	if err != nil {
		t.Fatalf("update container group returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected update container group status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeContainerInstanceResponse(t, updateResp)
	if updated["location"] != "eastus2" || updated["tags"].(map[string]any)["env"] != "prod" {
		t.Fatalf("expected update to replace location/tags, got %v", updated)
	}
	updatedProps := updated["properties"].(map[string]any)
	if updatedProps["restartPolicy"] != "OnFailure" {
		t.Fatalf("expected update to persist restart policy, got %v", updatedProps)
	}

	getResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, containerGroupURL, nil))
	if err != nil {
		t.Fatalf("get container group returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listRGResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list resource group container groups returned error: %v", err)
	}
	listRG := decodeContainerInstanceResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "cg-a" {
		t.Fatalf("expected one resource group container group, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/containerGroups?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list subscription container groups returned error: %v", err)
	}
	listSub := decodeContainerInstanceResponse(t, listSubResp)
	if values := listSub["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "cg-a" {
		t.Fatalf("expected one subscription container group, got %v", listSub)
	}

	restartResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-a/restart?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("restart container group returned error: %v", err)
	}
	if restartResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected restart status 204, got %d; body=%s", restartResp.StatusCode, string(restartResp.RawBody))
	}

	deleteResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodDelete, containerGroupURL, nil))
	if err != nil {
		t.Fatalf("delete container group returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected delete status 200, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}
}

func TestContainerGroupStartAndStopActionsUpdateInstanceView(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-actions?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload)); err != nil {
		t.Fatalf("create action container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create action container group status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	stopResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-actions/stop?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("stop container group returned error: %v", err)
	}
	if stopResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected stop status 204, got %d body=%s", stopResp.StatusCode, string(stopResp.RawBody))
	}

	stoppedResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, containerGroupURL, nil))
	if err != nil {
		t.Fatalf("get stopped container group returned error: %v", err)
	}
	stopped := decodeContainerInstanceResponse(t, stoppedResp)
	stoppedProps := stopped["properties"].(map[string]any)
	if stoppedProps["instanceView"].(map[string]any)["state"] != "Stopped" {
		t.Fatalf("expected stopped group instance view, got %v", stoppedProps["instanceView"])
	}
	stoppedContainers := stoppedProps["containers"].([]any)
	stoppedContainerView := stoppedContainers[0].(map[string]any)["properties"].(map[string]any)["instanceView"].(map[string]any)
	if stoppedContainerView["currentState"].(map[string]any)["state"] != "Terminated" {
		t.Fatalf("expected stopped container state to be Terminated, got %v", stoppedContainerView)
	}

	startResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-actions/start?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("start container group returned error: %v", err)
	}
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected start status 202, got %d body=%s", startResp.StatusCode, string(startResp.RawBody))
	}
	if startResp.Headers["Retry-After"] != "0" || startResp.Headers["Location"] == "" {
		t.Fatalf("expected start operation headers, got %v", startResp.Headers)
	}

	startedResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, containerGroupURL, nil))
	if err != nil {
		t.Fatalf("get started container group returned error: %v", err)
	}
	started := decodeContainerInstanceResponse(t, startedResp)
	startedProps := started["properties"].(map[string]any)
	if startedProps["instanceView"].(map[string]any)["state"] != "Running" {
		t.Fatalf("expected running group instance view, got %v", startedProps["instanceView"])
	}
	startedContainers := startedProps["containers"].([]any)
	startedContainerView := startedContainers[0].(map[string]any)["properties"].(map[string]any)["instanceView"].(map[string]any)
	if startedContainerView["currentState"].(map[string]any)["state"] != "Running" {
		t.Fatalf("expected started container state to be Running, got %v", startedContainerView)
	}
}

func TestContainerGroupContainerLogsHonorTailAndTimestamps(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-logs?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload)); err != nil {
		t.Fatalf("create log container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create log container group status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	logsURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-logs/containers/web/logs?api-version=2025-09-01&tail=1&timestamps=true"
	logsResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, logsURL, nil))
	if err != nil {
		t.Fatalf("list container logs returned error: %v", err)
	}
	if logsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected logs status 200, got %d body=%s", logsResp.StatusCode, string(logsResp.RawBody))
	}
	logs := decodeContainerInstanceResponse(t, logsResp)
	if logs["content"] != "2026-01-01T00:00:00Z image nginx:1.27 state Running" {
		t.Fatalf("unexpected timestamped tail logs: %v", logs)
	}

	missingResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-logs/containers/missing/logs?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("missing container logs returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing container logs status 404, got %d body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestContainerGroupContainerExecReturnsDeterministicWebSocketDetails(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-exec?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload)); err != nil {
		t.Fatalf("create exec container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create exec container group status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	execURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-exec/containers/web/exec?api-version=2025-09-01"
	execResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, execURL, []byte(`{"command":"/bin/bash","terminalSize":{"cols":120,"rows":40}}`)))
	if err != nil {
		t.Fatalf("exec container command returned error: %v", err)
	}
	if execResp.StatusCode != http.StatusOK {
		t.Fatalf("expected exec status 200, got %d body=%s", execResp.StatusCode, string(execResp.RawBody))
	}
	execBody := decodeContainerInstanceResponse(t, execResp)
	if execBody["password"] != "cloudmock-cg-exec-web-exec-token" {
		t.Fatalf("unexpected exec password: %v", execBody)
	}
	webSocketURI, ok := execBody["webSocketUri"].(string)
	if !ok || webSocketURI == "" {
		t.Fatalf("expected exec websocket URI, got %v", execBody)
	}
	parsedWebSocketURI, err := url.Parse(webSocketURI)
	if err != nil {
		t.Fatalf("parse exec websocket URI: %v", err)
	}
	if parsedWebSocketURI.Scheme != "wss" || parsedWebSocketURI.Host != "management.azure.com" ||
		parsedWebSocketURI.Path != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-exec/containers/web/exec/ws" {
		t.Fatalf("unexpected exec websocket URI: %s", webSocketURI)
	}
	if parsedWebSocketURI.Query().Get("command") != "/bin/bash" ||
		parsedWebSocketURI.Query().Get("cols") != "120" ||
		parsedWebSocketURI.Query().Get("rows") != "40" {
		t.Fatalf("unexpected exec websocket query: %s", webSocketURI)
	}

	missingResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-exec/containers/missing/exec?api-version=2025-09-01", []byte(`{"command":"/bin/bash"}`)))
	if err != nil {
		t.Fatalf("missing container exec returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing container exec status 404, got %d body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestContainerGroupContainerAttachReturnsDeterministicWebSocketDetails(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-attach?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload)); err != nil {
		t.Fatalf("create attach container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create attach container group status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	attachURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-attach/containers/web/attach?api-version=2025-09-01"
	attachResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, attachURL, nil))
	if err != nil {
		t.Fatalf("attach container returned error: %v", err)
	}
	if attachResp.StatusCode != http.StatusOK {
		t.Fatalf("expected attach status 200, got %d body=%s", attachResp.StatusCode, string(attachResp.RawBody))
	}
	attachBody := decodeContainerInstanceResponse(t, attachResp)
	if attachBody["password"] != "cloudmock-cg-attach-web-attach-token" {
		t.Fatalf("unexpected attach password: %v", attachBody)
	}
	webSocketURI, ok := attachBody["webSocketUri"].(string)
	if !ok || webSocketURI == "" {
		t.Fatalf("expected attach websocket URI, got %v", attachBody)
	}
	parsedWebSocketURI, err := url.Parse(webSocketURI)
	if err != nil {
		t.Fatalf("parse attach websocket URI: %v", err)
	}
	if parsedWebSocketURI.Scheme != "wss" || parsedWebSocketURI.Host != "management.azure.com" ||
		parsedWebSocketURI.Path != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-attach/containers/web/attach/ws" {
		t.Fatalf("unexpected attach websocket URI: %s", webSocketURI)
	}

	missingResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-attach/containers/missing/attach?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("missing container attach returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing container attach status 404, got %d body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestContainerGroupOutboundNetworkDependenciesEndpointsReturnsEmptyArray(t *testing.T) {
	svc := New()

	containerGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-network?api-version=2025-09-01"
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, containerGroupURL, payload)); err != nil {
		t.Fatalf("create network container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create network container group status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	dependenciesURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-network/outboundNetworkDependenciesEndpoints?api-version=2025-09-01"
	dependenciesResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, dependenciesURL, nil))
	if err != nil {
		t.Fatalf("get outbound network dependencies returned error: %v", err)
	}
	if dependenciesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected outbound network dependencies status 200, got %d body=%s", dependenciesResp.StatusCode, string(dependenciesResp.RawBody))
	}
	var dependencies []string
	if err := gojson.Unmarshal(dependenciesResp.RawBody, &dependencies); err != nil {
		t.Fatalf("decode outbound network dependencies: %v body=%s", err, string(dependenciesResp.RawBody))
	}
	if len(dependencies) != 0 {
		t.Fatalf("expected empty outbound network dependencies, got %#v", dependencies)
	}

	missingResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/missing/outboundNetworkDependenciesEndpoints?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("missing outbound network dependencies returned error: %v", err)
	}
	if missingResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing outbound network dependencies status 404, got %d body=%s", missingResp.StatusCode, string(missingResp.RawBody))
	}
}

func TestLocationListCachedImagesAndCapabilitiesReturnDocumentedEnvelopes(t *testing.T) {
	svc := New()

	cachedResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westcentralus/cachedImages?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list cached images returned error: %v", err)
	}
	if cachedResp.StatusCode != http.StatusOK {
		t.Fatalf("expected cached images status 200, got %d body=%s", cachedResp.StatusCode, string(cachedResp.RawBody))
	}
	cached := decodeContainerInstanceResponse(t, cachedResp)
	cachedValues := cached["value"].([]any)
	if len(cachedValues) < 3 {
		t.Fatalf("expected deterministic cached images, got %v", cached)
	}
	firstCached := cachedValues[0].(map[string]any)
	if firstCached["osType"] != "Linux" || firstCached["image"] == "" {
		t.Fatalf("expected cached image osType/image fields, got %v", firstCached)
	}

	capabilitiesResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westus/capabilities?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list capabilities returned error: %v", err)
	}
	if capabilitiesResp.StatusCode != http.StatusOK {
		t.Fatalf("expected capabilities status 200, got %d body=%s", capabilitiesResp.StatusCode, string(capabilitiesResp.RawBody))
	}
	capabilities := decodeContainerInstanceResponse(t, capabilitiesResp)
	capabilityValues := capabilities["value"].([]any)
	if len(capabilityValues) < 2 {
		t.Fatalf("expected deterministic capabilities, got %v", capabilities)
	}
	linuxCapability := capabilityValues[0].(map[string]any)
	if linuxCapability["location"] != "westus" || linuxCapability["resourceType"] != "containerGroups" || linuxCapability["osType"] != "Linux" || linuxCapability["ipAddressType"] != "Public" {
		t.Fatalf("unexpected capability identity fields: %v", linuxCapability)
	}
	limits := linuxCapability["capabilities"].(map[string]any)
	if limits["maxCpu"] != float64(4) || limits["maxMemoryInGB"] != float64(14) || limits["maxGpuCount"] != float64(4) {
		t.Fatalf("unexpected capability limits: %v", limits)
	}
}

func TestLocationListUsageCountsContainerGroupsInRegion(t *testing.T) {
	svc := New()

	payload := []byte(`{
		"location":"westcentralus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`)
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-usage-a?api-version=2025-09-01", payload)); err != nil {
		t.Fatalf("create first usage container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected first usage create status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-b/providers/Microsoft.ContainerInstance/containerGroups/cg-usage-b?api-version=2025-09-01", payload)); err != nil {
		t.Fatalf("create second usage container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected second usage create status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-other-region?api-version=2025-09-01", []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`))); err != nil {
		t.Fatalf("create other-region container group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected other-region create status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	usageResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/locations/westcentralus/usages?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list usage returned error: %v", err)
	}
	if usageResp.StatusCode != http.StatusOK {
		t.Fatalf("expected usage status 200, got %d body=%s", usageResp.StatusCode, string(usageResp.RawBody))
	}
	usage := decodeContainerInstanceResponse(t, usageResp)
	values := usage["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one usage record, got %v", usage)
	}
	record := values[0].(map[string]any)
	if record["currentValue"] != float64(2) || record["limit"] != float64(2000) || record["unit"] != "Count" {
		t.Fatalf("unexpected usage record values: %v", record)
	}
	name := record["name"].(map[string]any)
	if name["value"] != "ContainerGroups" || name["localizedValue"] != "Container Groups" {
		t.Fatalf("unexpected usage name: %v", name)
	}
}

func TestOperationsListReturnsProviderOperationMetadata(t *testing.T) {
	svc := New()

	operationsResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/providers/Microsoft.ContainerInstance/operations?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list operations returned error: %v", err)
	}
	if operationsResp.StatusCode != http.StatusOK {
		t.Fatalf("expected operations status 200, got %d body=%s", operationsResp.StatusCode, string(operationsResp.RawBody))
	}
	operations := decodeContainerInstanceResponse(t, operationsResp)
	values := operations["value"].([]any)
	if len(values) < 4 {
		t.Fatalf("expected provider operations metadata, got %v", operations)
	}

	byName := make(map[string]map[string]any, len(values))
	for _, value := range values {
		item := value.(map[string]any)
		byName[item["name"].(string)] = item
	}
	for _, name := range []string{
		"Microsoft.ContainerInstance/containerGroups/read",
		"Microsoft.ContainerInstance/containerGroups/write",
		"Microsoft.ContainerInstance/locations/cachedImages/read",
		"Microsoft.ContainerInstance/locations/usages/read",
	} {
		item, ok := byName[name]
		if !ok {
			t.Fatalf("expected operation %q in %v", name, byName)
		}
		if item["origin"] != "User" {
			t.Fatalf("expected operation %s origin User, got %v", name, item)
		}
		display := item["display"].(map[string]any)
		if display["provider"] != "Microsoft Container Instance" || display["resource"] == "" || display["operation"] == "" || display["description"] == "" {
			t.Fatalf("expected populated display for %s, got %v", name, display)
		}
	}
}

func TestContainerGroupProfileCreateGetUpdateAndPatchTags(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a?api-version=2025-09-01"
	createResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, profileURL, []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"zones":["1"],
		"properties":{
			"containers":[{
				"name":"web",
				"properties":{
					"image":"nginx:1.27",
					"resources":{"requests":{"cpu":1.0,"memoryInGB":1.5}},
					"ports":[{"port":80,"protocol":"TCP"}]
				}
			}],
			"imageRegistryCredentials":[{"server":"index.docker.io","username":"cloudmock"}],
			"ipAddress":{"type":"Public","ports":[{"port":80,"protocol":"TCP"}]},
			"osType":"Linux",
			"restartPolicy":"Never",
			"sku":"Standard",
			"volumes":[{"name":"scratch","emptyDir":{}}]
		}
	}`)))
	if err != nil {
		t.Fatalf("create container group profile returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected profile create status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeContainerInstanceResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-a" {
		t.Fatalf("unexpected profile id: %v", created["id"])
	}
	if created["name"] != "cgp-a" || created["type"] != "Microsoft.ContainerInstance/containerGroupProfiles" || created["location"] != "eastus" {
		t.Fatalf("unexpected profile identity fields: %v", created)
	}
	if created["tags"].(map[string]any)["env"] != "test" {
		t.Fatalf("expected create tags to be preserved, got %v", created["tags"])
	}
	if zones := created["zones"].([]any); len(zones) != 1 || zones[0] != "1" {
		t.Fatalf("expected create zones to be preserved, got %v", created["zones"])
	}
	createdProps := created["properties"].(map[string]any)
	if createdProps["revision"] != float64(1) {
		t.Fatalf("expected first profile revision 1, got %v", createdProps["revision"])
	}
	if createdProps["osType"] != "Linux" || createdProps["sku"] != "Standard" {
		t.Fatalf("expected create properties to be preserved, got %v", createdProps)
	}

	updateResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, profileURL, []byte(`{
		"location":"eastus2",
		"tags":{"env":"prod"},
		"properties":{
			"containers":[{"name":"worker","properties":{"image":"alpine:3.20","resources":{"requests":{"cpu":0.5,"memoryInGB":1.0}}}}],
			"osType":"Linux",
			"restartPolicy":"OnFailure"
		}
	}`)))
	if err != nil {
		t.Fatalf("update container group profile returned error: %v", err)
	}
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile update status 200, got %d; body=%s", updateResp.StatusCode, string(updateResp.RawBody))
	}
	updated := decodeContainerInstanceResponse(t, updateResp)
	updatedProps := updated["properties"].(map[string]any)
	if updated["location"] != "eastus2" || updated["tags"].(map[string]any)["env"] != "prod" {
		t.Fatalf("expected profile update to replace location/tags, got %v", updated)
	}
	if updatedProps["revision"] != float64(2) {
		t.Fatalf("expected second profile revision 2, got %v", updatedProps["revision"])
	}
	if revisions := updatedProps["registeredRevisions"].([]any); len(revisions) != 2 || revisions[0] != float64(1) || revisions[1] != float64(2) {
		t.Fatalf("expected registered revisions [1 2], got %v", updatedProps["registeredRevisions"])
	}

	patchResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPatch, profileURL, []byte(`{"tags":{"team":"platform"}}`)))
	if err != nil {
		t.Fatalf("patch container group profile returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile patch status 200, got %d; body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	patched := decodeContainerInstanceResponse(t, patchResp)
	if patched["tags"].(map[string]any)["team"] != "platform" {
		t.Fatalf("expected profile patch tags to be stored, got %v", patched["tags"])
	}
	if patched["properties"].(map[string]any)["revision"] != float64(2) {
		t.Fatalf("expected tag patch to preserve current revision, got %v", patched["properties"])
	}

	getResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, profileURL, nil))
	if err != nil {
		t.Fatalf("get container group profile returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile get status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	got := decodeContainerInstanceResponse(t, getResp)
	if got["tags"].(map[string]any)["team"] != "platform" || got["properties"].(map[string]any)["revision"] != float64(2) {
		t.Fatalf("expected get to return patched current profile, got %v", got)
	}
}

func TestContainerGroupProfileListsByResourceGroupAndSubscription(t *testing.T) {
	svc := New()
	createContainerGroupProfileForTest(t, svc, "sub-1", "rg-a", "cgp-a", "eastus", "web")
	createContainerGroupProfileForTest(t, svc, "sub-1", "rg-b", "cgp-b", "westus", "worker")
	createContainerGroupProfileForTest(t, svc, "sub-2", "rg-a", "cgp-other", "eastus", "other")

	listRGResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list resource group profiles returned error: %v", err)
	}
	if listRGResp.StatusCode != http.StatusOK {
		t.Fatalf("expected resource group profile list status 200, got %d; body=%s", listRGResp.StatusCode, string(listRGResp.RawBody))
	}
	listRG := decodeContainerInstanceResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "cgp-a" {
		t.Fatalf("expected one rg profile, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.ContainerInstance/containerGroupProfiles?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list subscription profiles returned error: %v", err)
	}
	if listSubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription profile list status 200, got %d; body=%s", listSubResp.StatusCode, string(listSubResp.RawBody))
	}
	listSub := decodeContainerInstanceResponse(t, listSubResp)
	values := listSub["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "cgp-a" || values[1].(map[string]any)["name"] != "cgp-b" {
		t.Fatalf("expected two sorted subscription profiles, got %v", listSub)
	}
}

func TestContainerGroupProfileRevisionListAndGetByRevisionNumber(t *testing.T) {
	svc := New()

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-rev?api-version=2025-09-01"
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, profileURL, []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"web","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1,"memoryInGB":1.5}}}}],
			"osType":"Linux"
		}
	}`))); err != nil {
		t.Fatalf("create revision profile returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create revision profile status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	if resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, profileURL, []byte(`{
		"location":"eastus",
		"properties":{
			"containers":[{"name":"worker","properties":{"image":"alpine:3.20","resources":{"requests":{"cpu":0.5,"memoryInGB":1}}}}],
			"osType":"Linux"
		}
	}`))); err != nil {
		t.Fatalf("update revision profile returned error: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected update revision profile status 200, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	listResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-rev/revisions?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("list profile revisions returned error: %v", err)
	}
	if listResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile revisions list status 200, got %d body=%s", listResp.StatusCode, string(listResp.RawBody))
	}
	revisions := decodeContainerInstanceResponse(t, listResp)["value"].([]any)
	if len(revisions) != 2 {
		t.Fatalf("expected two revisions, got %v", revisions)
	}
	firstProps := revisions[0].(map[string]any)["properties"].(map[string]any)
	secondProps := revisions[1].(map[string]any)["properties"].(map[string]any)
	if firstProps["revision"] != float64(1) || secondProps["revision"] != float64(2) {
		t.Fatalf("expected revisions 1 and 2, got %v", revisions)
	}

	getRevisionResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-rev/revisions/1?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("get profile revision returned error: %v", err)
	}
	if getRevisionResp.StatusCode != http.StatusOK {
		t.Fatalf("expected profile revision get status 200, got %d body=%s", getRevisionResp.StatusCode, string(getRevisionResp.RawBody))
	}
	revisionOne := decodeContainerInstanceResponse(t, getRevisionResp)
	revisionOneProps := revisionOne["properties"].(map[string]any)
	containers := revisionOneProps["containers"].([]any)
	if revisionOneProps["revision"] != float64(1) || containers[0].(map[string]any)["name"] != "web" {
		t.Fatalf("expected revision 1 to preserve first profile version, got %v", revisionOne)
	}

	missingRevisionResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-rev/revisions/99?api-version=2025-09-01", nil))
	if err != nil {
		t.Fatalf("missing profile revision returned error: %v", err)
	}
	if missingRevisionResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing profile revision status 404, got %d body=%s", missingRevisionResp.StatusCode, string(missingRevisionResp.RawBody))
	}
}

func TestContainerGroupProfileDeleteMatchesAzureStatusCodes(t *testing.T) {
	svc := New()
	createContainerGroupProfileForTest(t, svc, "sub-1", "rg-a", "cgp-delete", "eastus", "web")

	profileURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-delete?api-version=2025-09-01"
	deleteResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodDelete, profileURL, nil))
	if err != nil {
		t.Fatalf("delete container group profile returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected existing profile delete status 200, got %d body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	getResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodGet, profileURL, nil))
	if err != nil {
		t.Fatalf("get deleted container group profile returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted profile get status 404, got %d body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	missingDeleteResp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodDelete, profileURL, nil))
	if err != nil {
		t.Fatalf("delete missing container group profile returned error: %v", err)
	}
	if missingDeleteResp.StatusCode != http.StatusNoContent {
		t.Fatalf("expected missing profile delete status 204, got %d body=%s", missingDeleteResp.StatusCode, string(missingDeleteResp.RawBody))
	}
}

func TestContainerGroupProfileValidationTemplateProvisioningAndKeys(t *testing.T) {
	svc := New()

	badURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/bad?api-version=2025-09-01"
	missingContainers, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, badURL, []byte(`{"location":"eastus","properties":{"osType":"Linux"}}`)))
	if err != nil {
		t.Fatalf("missing profile containers request returned error: %v", err)
	}
	if missingContainers.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing profile containers status 400, got %d; body=%s", missingContainers.StatusCode, string(missingContainers.RawBody))
	}

	result, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.ContainerInstance/containerGroupProfiles",
		"name":     "cgp-template",
		"location": "westus",
		"tags":     map[string]any{"source": "template"},
		"zones":    []any{"1"},
		"properties": map[string]any{
			"containers": []any{
				map[string]any{
					"name": "job",
					"properties": map[string]any{
						"image":     "busybox",
						"resources": map[string]any{"requests": map[string]any{"cpu": 1, "memoryInGB": 1}},
					},
				},
			},
			"osType":        "Linux",
			"restartPolicy": "Never",
		},
	})
	if err != nil {
		t.Fatalf("provision container group profile returned error: %v", err)
	}
	profile := result.(map[string]any)
	if profile["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroupProfiles/cgp-template" {
		t.Fatalf("unexpected provisioned profile id: %v", profile["id"])
	}
	if profile["type"] != "Microsoft.ContainerInstance/containerGroupProfiles" || profile["properties"].(map[string]any)["revision"] != float64(1) {
		t.Fatalf("unexpected provisioned profile: %v", profile)
	}

	want := routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerInstance/containerGroupProfiles", APIVersion: "2025-09-01"}
	for _, got := range svc.ServiceKeys() {
		if got == want {
			return
		}
	}
	t.Fatalf("expected profile service key %#v in %#v", want, svc.ServiceKeys())
}

func TestContainerGroupValidationTemplateProvisioningAndKeys(t *testing.T) {
	svc := New()

	badURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/bad?api-version=2025-09-01"
	missingContainers, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, badURL, []byte(`{"location":"eastus","properties":{"osType":"Linux"}}`)))
	if err != nil {
		t.Fatalf("missing containers request returned error: %v", err)
	}
	if missingContainers.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing containers status 400, got %d; body=%s", missingContainers.StatusCode, string(missingContainers.RawBody))
	}

	result, err := svc.ProvisionTemplateResource("sub-1", "rg-a", map[string]any{
		"type":     "Microsoft.ContainerInstance/containerGroups",
		"name":     "cg-template",
		"location": "westus",
		"tags":     map[string]any{"source": "template"},
		"properties": map[string]any{
			"containers": []any{
				map[string]any{
					"name": "job",
					"properties": map[string]any{
						"image":     "busybox",
						"resources": map[string]any{"requests": map[string]any{"cpu": 1, "memoryInGB": 1}},
					},
				},
			},
			"osType":        "Linux",
			"restartPolicy": "Never",
		},
	})
	if err != nil {
		t.Fatalf("provision container group returned error: %v", err)
	}
	containerGroup := result.(map[string]any)
	if containerGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.ContainerInstance/containerGroups/cg-template" {
		t.Fatalf("unexpected provisioned container group id: %v", containerGroup["id"])
	}
	if containerGroup["type"] != "Microsoft.ContainerInstance/containerGroups" {
		t.Fatalf("unexpected provisioned container group type: %v", containerGroup["type"])
	}

	keys := svc.ServiceKeys()
	expected := []routing.ServiceKey{
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerInstance/containerGroups", APIVersion: "2025-09-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerInstance/locations", APIVersion: "2025-09-01"},
		{Provider: routing.ProviderAzure, Service: "Microsoft.ContainerInstance/operations", APIVersion: "2025-09-01"},
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

func createContainerGroupProfileForTest(t *testing.T, svc *ContainerInstanceService, subscriptionID, resourceGroup, name, location, containerName string) {
	t.Helper()
	rawURL := "https://management.azure.com/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.ContainerInstance/containerGroupProfiles/" + name + "?api-version=2025-09-01"
	body := []byte(`{
		"location":"` + location + `",
		"properties":{
			"containers":[{"name":"` + containerName + `","properties":{"image":"nginx:1.27","resources":{"requests":{"cpu":1,"memoryInGB":1}}}}],
			"osType":"Linux"
		}
	}`)
	resp, err := svc.HandleRequest(containerInstanceCtx(t, http.MethodPut, rawURL, body))
	if err != nil {
		t.Fatalf("create test profile %s returned error: %v", name, err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create test profile %s status 201, got %d body=%s", name, resp.StatusCode, string(resp.RawBody))
	}
}

func containerInstanceCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
	t.Helper()
	parsed, err := url.Parse(rawURL)
	if err != nil {
		t.Fatalf("parse url: %v", err)
	}
	req := &http.Request{
		Method: method,
		URL:    parsed,
		Host:   parsed.Host,
		Header: make(http.Header),
	}
	req.Header.Set("Authorization", "Bearer token")
	return &service.RequestContext{
		Service:    "Microsoft.ContainerInstance/containerGroups",
		Action:     routing.DetectTarget(req).Action,
		RawRequest: req,
		Body:       body,
	}
}

func decodeContainerInstanceResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
