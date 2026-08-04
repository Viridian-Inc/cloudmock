package compute_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/azure/compute"
)

func computeCtx(t *testing.T, method, targetURL string, body []byte) *service.RequestContext {
	t.Helper()

	req := httptest.NewRequest(method, targetURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer test-token")
	return &service.RequestContext{
		AccountID:  "00000000-0000-0000-0000-000000000001",
		RawRequest: req,
		Body:       body,
	}
}

func decodeComputeResponse(t *testing.T, resp *service.Response) map[string]any {
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

func TestVirtualMachineLifecycle(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_D2s_v5"},
			"osProfile":{"computerName":"vm-a","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a","properties":{"primary":true}}]}
		}
	}`)
	vmURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01"

	createResp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, vmURL, payload))
	if err != nil {
		t.Fatalf("create VM returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create VM status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeComputeResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a" {
		t.Fatalf("unexpected VM id: %v", created["id"])
	}
	if created["name"] != "vm-a" || created["type"] != "Microsoft.Compute/virtualMachines" || created["location"] != "eastus" {
		t.Fatalf("unexpected VM identity fields: %v", created)
	}
	props := created["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected provisioningState Succeeded, got %v", props["provisioningState"])
	}
	hardware := props["hardwareProfile"].(map[string]any)
	if hardware["vmSize"] != "Standard_D2s_v5" {
		t.Fatalf("unexpected hardware profile: %v", hardware)
	}

	getResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, vmURL, nil))
	if err != nil {
		t.Fatalf("get VM returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get VM status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("list VMs returned error: %v", err)
	}
	listed := decodeComputeResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one VM in list, got %d in %v", len(values), listed)
	}

	deleteResp, err := svc.HandleRequest(computeCtx(t, http.MethodDelete, vmURL, nil))
	if err != nil {
		t.Fatalf("delete VM returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete VM status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	getDeletedResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, vmURL, nil))
	if err != nil {
		t.Fatalf("get deleted VM returned error: %v", err)
	}
	if getDeletedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted VM status 404, got %d; body=%s", getDeletedResp.StatusCode, string(getDeletedResp.RawBody))
	}
}

func TestVirtualMachineGetExpandsInstanceView(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_D2s_v5"},
			"osProfile":{"computerName":"vm-a","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a"}]}
		}
	}`)
	vmURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01"
	_, err := svc.HandleRequest(computeCtx(t, http.MethodPut, vmURL, payload))
	if err != nil {
		t.Fatalf("create VM returned error: %v", err)
	}

	getResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, vmURL+"&$expand=instanceView", nil))
	if err != nil {
		t.Fatalf("get VM instance view returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get VM status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}
	body := decodeComputeResponse(t, getResp)
	props := body["properties"].(map[string]any)
	instanceView, ok := props["instanceView"].(map[string]any)
	if !ok {
		t.Fatalf("expected properties.instanceView in expanded response, got %v", props)
	}
	statuses := instanceView["statuses"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("expected provisioning and power statuses, got %v", statuses)
	}
	first := statuses[0].(map[string]any)
	second := statuses[1].(map[string]any)
	if first["code"] != "ProvisioningState/succeeded" || first["displayStatus"] != "Provisioning succeeded" {
		t.Fatalf("unexpected provisioning status: %v", first)
	}
	if second["code"] != "PowerState/running" || second["displayStatus"] != "VM running" {
		t.Fatalf("unexpected power status: %v", second)
	}
}

func TestVirtualMachineStartAndDeallocateUpdateInstanceViewPowerState(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_D2s_v5"},
			"osProfile":{"computerName":"vm-a","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a"}]}
		}
	}`)
	vmURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a?api-version=2025-11-01"
	_, err := svc.HandleRequest(computeCtx(t, http.MethodPut, vmURL, payload))
	if err != nil {
		t.Fatalf("create VM returned error: %v", err)
	}

	deallocateResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/deallocate?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("deallocate VM returned error: %v", err)
	}
	if deallocateResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected deallocate status 202, got %d; body=%s", deallocateResp.StatusCode, string(deallocateResp.RawBody))
	}
	assertPowerState(t, svc, vmURL, "PowerState/deallocated", "VM deallocated")

	startResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-a/start?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("start VM returned error: %v", err)
	}
	if startResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected start status 202, got %d; body=%s", startResp.StatusCode, string(startResp.RawBody))
	}
	assertPowerState(t, svc, vmURL, "PowerState/running", "VM running")
}

func TestVirtualMachinePatchUpdatesTagsAndProperties(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_D2s_v5"},
			"osProfile":{"computerName":"vm-patch","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a"}]}
		}
	}`)
	vmURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-patch?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, vmURL, payload)); err != nil {
		t.Fatalf("create patch VM returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create patch VM status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	patchResp, err := svc.HandleRequest(computeCtx(t, http.MethodPatch, vmURL, []byte(`{
		"tags":{"team":"platform"},
		"properties":{"diagnosticsProfile":{"bootDiagnostics":{"enabled":true}}}
	}`)))
	if err != nil {
		t.Fatalf("patch VM returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch VM status 200, got %d body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	patched := decodeComputeResponse(t, patchResp)
	if patched["location"] != "eastus" || patched["tags"].(map[string]any)["team"] != "platform" {
		t.Fatalf("expected patch to preserve location and replace tags, got %v", patched)
	}
	props := patched["properties"].(map[string]any)
	if props["hardwareProfile"].(map[string]any)["vmSize"] != "Standard_D2s_v5" {
		t.Fatalf("expected patch to preserve existing hardware profile, got %v", props)
	}
	bootDiagnostics := props["diagnosticsProfile"].(map[string]any)["bootDiagnostics"].(map[string]any)
	if bootDiagnostics["enabled"] != true {
		t.Fatalf("expected patch to merge diagnostics profile, got %v", props)
	}
}

func TestVirtualMachineInstanceViewRouteAndAdditionalPowerActions(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_D2s_v5"},
			"osProfile":{"computerName":"vm-power","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a"}]}
		}
	}`)
	baseURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-power"
	vmURL := baseURL + "?api-version=2025-11-01"
	if resp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, vmURL, payload)); err != nil {
		t.Fatalf("create power VM returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create power VM status 201, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}

	assertInstanceViewRoute(t, svc, baseURL, "PowerState/running", "VM running")

	powerOffResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, baseURL+"/powerOff?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("powerOff VM returned error: %v", err)
	}
	if powerOffResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected powerOff status 202, got %d body=%s", powerOffResp.StatusCode, string(powerOffResp.RawBody))
	}
	assertInstanceViewRoute(t, svc, baseURL, "PowerState/stopped", "VM stopped")

	for _, operation := range []string{"restart", "redeploy", "reapply"} {
		resp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, baseURL+"/"+operation+"?api-version=2025-11-01", nil))
		if err != nil {
			t.Fatalf("%s VM returned error: %v", operation, err)
		}
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("expected %s status 202, got %d body=%s", operation, resp.StatusCode, string(resp.RawBody))
		}
		assertInstanceViewRoute(t, svc, baseURL, "PowerState/running", "VM running")
	}
}

func TestVirtualMachineListsByResourceGroupAndSubscription(t *testing.T) {
	svc := compute.New()
	createVirtualMachineForTest(t, svc, "sub-1", "rg-a", "vm-a")
	createVirtualMachineForTest(t, svc, "sub-1", "rg-b", "vm-b")
	createVirtualMachineForTest(t, svc, "sub-2", "rg-a", "vm-other")

	listRGResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("list resource group VMs returned error: %v", err)
	}
	listRG := decodeComputeResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "vm-a" {
		t.Fatalf("expected one resource group VM, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Compute/virtualMachines?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("list subscription VMs returned error: %v", err)
	}
	if listSubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription VM list status 200, got %d body=%s", listSubResp.StatusCode, string(listSubResp.RawBody))
	}
	listSub := decodeComputeResponse(t, listSubResp)
	values := listSub["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "vm-a" || values[1].(map[string]any)["name"] != "vm-b" {
		t.Fatalf("expected two sorted subscription VMs, got %v", listSub)
	}
}

func TestVirtualMachineRunCommandReturnsDocumentedStatusRecords(t *testing.T) {
	svc := compute.New()
	createVirtualMachineForTest(t, svc, "sub-1", "rg-a", "vm-run")

	runCommandURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-run/runCommand?api-version=2025-11-01"
	resp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, runCommandURL, []byte(`{
		"commandId":"RunShellScript",
		"script":["echo hello","uname -a"],
		"parameters":[{"name":"who","value":"cloudmock"},{"name":"mode","value":"local"}]
	}`)))
	if err != nil {
		t.Fatalf("run command returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected run command status 200, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeComputeResponse(t, resp)
	values := body["value"].([]any)
	if len(values) != 2 {
		t.Fatalf("expected stdout and stderr status records, got %v", body)
	}
	stdout := values[0].(map[string]any)
	if stdout["code"] != "ComponentStatus/StdOut/succeeded" || stdout["level"] != "Info" || stdout["displayStatus"] != "Provisioning succeeded" {
		t.Fatalf("unexpected stdout status metadata: %v", stdout)
	}
	if stdout["message"] != "RunShellScript executed on vm-run\nscript:\necho hello\nuname -a\nparameters:\nwho=cloudmock\nmode=local" {
		t.Fatalf("unexpected stdout message: %q", stdout["message"])
	}
	stderr := values[1].(map[string]any)
	if stderr["code"] != "ComponentStatus/StdErr/succeeded" || stderr["level"] != "Info" || stderr["displayStatus"] != "Provisioning succeeded" || stderr["message"] != "" {
		t.Fatalf("unexpected stderr status: %v", stderr)
	}
}

func TestVirtualMachineRunCommandValidationAndMissingVM(t *testing.T) {
	svc := compute.New()
	createVirtualMachineForTest(t, svc, "sub-1", "rg-a", "vm-run")

	runCommandURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/vm-run/runCommand?api-version=2025-11-01"
	missingCommandIDResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, runCommandURL, []byte(`{"script":["echo missing"]}`)))
	if err != nil {
		t.Fatalf("missing commandId request returned error: %v", err)
	}
	if missingCommandIDResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing commandId status 400, got %d body=%s", missingCommandIDResp.StatusCode, string(missingCommandIDResp.RawBody))
	}

	missingVMResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/virtualMachines/missing/runCommand?api-version=2025-11-01", []byte(`{"commandId":"RunShellScript"}`)))
	if err != nil {
		t.Fatalf("missing VM run command returned error: %v", err)
	}
	if missingVMResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing VM run command status 404, got %d body=%s", missingVMResp.StatusCode, string(missingVMResp.RawBody))
	}
}

func assertPowerState(t *testing.T, svc *compute.ComputeService, vmURL, code, display string) {
	t.Helper()

	resp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, vmURL+"&$expand=instanceView", nil))
	if err != nil {
		t.Fatalf("get VM instance view returned error: %v", err)
	}
	body := decodeComputeResponse(t, resp)
	props := body["properties"].(map[string]any)
	instanceView := props["instanceView"].(map[string]any)
	statuses := instanceView["statuses"].([]any)
	power := statuses[1].(map[string]any)
	if power["code"] != code || power["displayStatus"] != display {
		t.Fatalf("unexpected power status: %v", power)
	}
}

func assertInstanceViewRoute(t *testing.T, svc *compute.ComputeService, baseURL, code, display string) {
	t.Helper()

	resp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, baseURL+"/instanceView?api-version=2025-11-01", nil))
	if err != nil {
		t.Fatalf("get VM instanceView route returned error: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected instanceView route status 200, got %d body=%s", resp.StatusCode, string(resp.RawBody))
	}
	body := decodeComputeResponse(t, resp)
	statuses := body["statuses"].([]any)
	if len(statuses) != 2 {
		t.Fatalf("expected provisioning and power statuses, got %v", statuses)
	}
	power := statuses[1].(map[string]any)
	if power["code"] != code || power["displayStatus"] != display {
		t.Fatalf("unexpected instanceView route power status: %v", power)
	}
}

func createVirtualMachineForTest(t *testing.T, svc *compute.ComputeService, subscriptionID, resourceGroup, name string) {
	t.Helper()
	rawURL := "https://management.azure.com/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/virtualMachines/" + name + "?api-version=2025-11-01"
	body := []byte(`{
		"location":"eastus",
		"properties":{
			"hardwareProfile":{"vmSize":"Standard_B1s"},
			"osProfile":{"computerName":"` + name + `","adminUsername":"azureuser"},
			"storageProfile":{"imageReference":{"publisher":"Canonical","offer":"0001-com-ubuntu-server-jammy","sku":"22_04-lts","version":"latest"}},
			"networkProfile":{"networkInterfaces":[{"id":"/subscriptions/` + subscriptionID + `/resourceGroups/` + resourceGroup + `/providers/Microsoft.Network/networkInterfaces/nic-` + name + `"}]}
		}
	}`)
	resp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, rawURL, body))
	if err != nil {
		t.Fatalf("create test VM %s returned error: %v", name, err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create test VM %s status 201, got %d body=%s", name, resp.StatusCode, string(resp.RawBody))
	}
}

func TestManagedDiskLifecycle(t *testing.T) {
	svc := compute.New()
	payload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Premium_LRS"},
		"tags":{"env":"test"},
		"properties":{
			"creationData":{"createOption":"Empty"},
			"diskSizeGB":128,
			"osType":"Linux"
		}
	}`)
	diskURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a?api-version=2025-01-02"

	createResp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, diskURL, payload))
	if err != nil {
		t.Fatalf("create disk returned error: %v", err)
	}
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create disk status 201, got %d; body=%s", createResp.StatusCode, string(createResp.RawBody))
	}
	created := decodeComputeResponse(t, createResp)
	if created["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-a" {
		t.Fatalf("unexpected disk id: %v", created["id"])
	}
	if created["name"] != "disk-a" || created["type"] != "Microsoft.Compute/disks" || created["location"] != "eastus" {
		t.Fatalf("unexpected disk identity fields: %v", created)
	}
	sku := created["sku"].(map[string]any)
	if sku["name"] != "Premium_LRS" {
		t.Fatalf("unexpected disk sku: %v", sku)
	}
	props := created["properties"].(map[string]any)
	if props["provisioningState"] != "Succeeded" || props["diskState"] != "Unattached" {
		t.Fatalf("unexpected disk state properties: %v", props)
	}
	if props["diskSizeGB"].(float64) != 128 {
		t.Fatalf("unexpected diskSizeGB: %v", props["diskSizeGB"])
	}

	getResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, diskURL, nil))
	if err != nil {
		t.Fatalf("get disk returned error: %v", err)
	}
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get disk status 200, got %d; body=%s", getResp.StatusCode, string(getResp.RawBody))
	}

	listResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks?api-version=2025-01-02", nil))
	if err != nil {
		t.Fatalf("list disks returned error: %v", err)
	}
	listed := decodeComputeResponse(t, listResp)
	values := listed["value"].([]any)
	if len(values) != 1 {
		t.Fatalf("expected one disk in list, got %d in %v", len(values), listed)
	}

	deleteResp, err := svc.HandleRequest(computeCtx(t, http.MethodDelete, diskURL, nil))
	if err != nil {
		t.Fatalf("delete disk returned error: %v", err)
	}
	if deleteResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete disk status 202, got %d; body=%s", deleteResp.StatusCode, string(deleteResp.RawBody))
	}

	getDeletedResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, diskURL, nil))
	if err != nil {
		t.Fatalf("get deleted disk returned error: %v", err)
	}
	if getDeletedResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected deleted disk status 404, got %d; body=%s", getDeletedResp.StatusCode, string(getDeletedResp.RawBody))
	}
}

func TestManagedDiskPatchUpdatesTagsSKUAndProperties(t *testing.T) {
	svc := compute.New()
	createManagedDiskForTest(t, svc, "sub-1", "rg-a", "disk-patch", 128)

	diskURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-patch?api-version=2025-01-02"
	patchResp, err := svc.HandleRequest(computeCtx(t, http.MethodPatch, diskURL, []byte(`{
		"tags":{"team":"storage"},
		"sku":{"name":"StandardSSD_LRS"},
		"properties":{"diskSizeGB":256,"networkAccessPolicy":"DenyAll"}
	}`)))
	if err != nil {
		t.Fatalf("patch disk returned error: %v", err)
	}
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("expected patch disk status 200, got %d body=%s", patchResp.StatusCode, string(patchResp.RawBody))
	}
	patched := decodeComputeResponse(t, patchResp)
	if patched["tags"].(map[string]any)["team"] != "storage" {
		t.Fatalf("expected patch tags to be stored, got %v", patched["tags"])
	}
	if patched["sku"].(map[string]any)["name"] != "StandardSSD_LRS" {
		t.Fatalf("expected patch SKU to be stored, got %v", patched["sku"])
	}
	props := patched["properties"].(map[string]any)
	if props["diskSizeGB"] != float64(256) || props["networkAccessPolicy"] != "DenyAll" {
		t.Fatalf("expected patch properties to be merged, got %v", props)
	}
	if props["diskState"] != "Unattached" || props["provisioningState"] != "Succeeded" {
		t.Fatalf("expected patch to preserve deterministic state, got %v", props)
	}
	if props["creationData"].(map[string]any)["createOption"] != "Empty" {
		t.Fatalf("expected patch to preserve creation data, got %v", props)
	}
}

func TestManagedDiskListsByResourceGroupAndSubscription(t *testing.T) {
	svc := compute.New()
	createManagedDiskForTest(t, svc, "sub-1", "rg-a", "disk-a", 64)
	createManagedDiskForTest(t, svc, "sub-1", "rg-b", "disk-b", 128)
	createManagedDiskForTest(t, svc, "sub-2", "rg-a", "disk-other", 256)

	listRGResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks?api-version=2025-01-02", nil))
	if err != nil {
		t.Fatalf("list resource group disks returned error: %v", err)
	}
	listRG := decodeComputeResponse(t, listRGResp)
	if values := listRG["value"].([]any); len(values) != 1 || values[0].(map[string]any)["name"] != "disk-a" {
		t.Fatalf("expected one resource group disk, got %v", listRG)
	}

	listSubResp, err := svc.HandleRequest(computeCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/providers/Microsoft.Compute/disks?api-version=2025-01-02", nil))
	if err != nil {
		t.Fatalf("list subscription disks returned error: %v", err)
	}
	if listSubResp.StatusCode != http.StatusOK {
		t.Fatalf("expected subscription disk list status 200, got %d body=%s", listSubResp.StatusCode, string(listSubResp.RawBody))
	}
	listSub := decodeComputeResponse(t, listSubResp)
	values := listSub["value"].([]any)
	if len(values) != 2 || values[0].(map[string]any)["name"] != "disk-a" || values[1].(map[string]any)["name"] != "disk-b" {
		t.Fatalf("expected two sorted subscription disks, got %v", listSub)
	}
}

func TestManagedDiskGrantAndRevokeAccess(t *testing.T) {
	svc := compute.New()
	createManagedDiskForTest(t, svc, "sub-1", "rg-a", "disk-access", 64)

	grantURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-access/beginGetAccess?api-version=2025-01-02"
	grantResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, grantURL, []byte(`{
		"access":"Read",
		"durationInSeconds":300,
		"fileFormat":"VHD",
		"getSecureVMGuestStateSAS":true
	}`)))
	if err != nil {
		t.Fatalf("grant disk access returned error: %v", err)
	}
	if grantResp.StatusCode != http.StatusOK {
		t.Fatalf("expected grant access status 200, got %d body=%s", grantResp.StatusCode, string(grantResp.RawBody))
	}
	granted := decodeComputeResponse(t, grantResp)
	accessSAS, ok := granted["accessSAS"].(string)
	if !ok || !strings.Contains(accessSAS, "/disk-access?") || !strings.Contains(accessSAS, "sp=r") || !strings.Contains(accessSAS, "se=300") {
		t.Fatalf("expected deterministic read SAS with duration, got %v", granted)
	}
	if securityData, ok := granted["securityDataAccessSAS"].(string); !ok || !strings.Contains(securityData, "_vmgs?") {
		t.Fatalf("expected VM guest state SAS when requested, got %v", granted)
	}
	if securityMetadata, ok := granted["securityMetadataAccessSAS"].(string); !ok || !strings.Contains(securityMetadata, "_vmmd?") {
		t.Fatalf("expected VM metadata SAS when requested, got %v", granted)
	}

	revokeResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-access/endGetAccess?api-version=2025-01-02", nil))
	if err != nil {
		t.Fatalf("revoke disk access returned error: %v", err)
	}
	if revokeResp.StatusCode != http.StatusOK || len(revokeResp.RawBody) != 0 {
		t.Fatalf("expected revoke access status 200 with empty body, got %d body=%s", revokeResp.StatusCode, string(revokeResp.RawBody))
	}
}

func TestManagedDiskGrantAccessValidationAndMissingDisk(t *testing.T) {
	svc := compute.New()
	createManagedDiskForTest(t, svc, "sub-1", "rg-a", "disk-access", 64)

	grantURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/disk-access/beginGetAccess?api-version=2025-01-02"
	missingAccessResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, grantURL, []byte(`{"durationInSeconds":300}`)))
	if err != nil {
		t.Fatalf("missing access grant request returned error: %v", err)
	}
	if missingAccessResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing access status 400, got %d body=%s", missingAccessResp.StatusCode, string(missingAccessResp.RawBody))
	}

	missingDurationResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, grantURL, []byte(`{"access":"Write"}`)))
	if err != nil {
		t.Fatalf("missing duration grant request returned error: %v", err)
	}
	if missingDurationResp.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected missing duration status 400, got %d body=%s", missingDurationResp.StatusCode, string(missingDurationResp.RawBody))
	}

	missingDiskResp, err := svc.HandleRequest(computeCtx(t, http.MethodPost, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Compute/disks/missing/beginGetAccess?api-version=2025-01-02", []byte(`{"access":"Read","durationInSeconds":300}`)))
	if err != nil {
		t.Fatalf("missing disk grant request returned error: %v", err)
	}
	if missingDiskResp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected missing disk grant status 404, got %d body=%s", missingDiskResp.StatusCode, string(missingDiskResp.RawBody))
	}
}

func createManagedDiskForTest(t *testing.T, svc *compute.ComputeService, subscriptionID, resourceGroup, name string, sizeGB int) {
	t.Helper()
	rawURL := "https://management.azure.com/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Compute/disks/" + name + "?api-version=2025-01-02"
	body := []byte(`{
		"location":"eastus",
		"sku":{"name":"Premium_LRS"},
		"properties":{"creationData":{"createOption":"Empty"},"diskSizeGB":` + fmt.Sprintf("%d", sizeGB) + `}
	}`)
	resp, err := svc.HandleRequest(computeCtx(t, http.MethodPut, rawURL, body))
	if err != nil {
		t.Fatalf("create test disk %s returned error: %v", name, err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create test disk %s status 201, got %d body=%s", name, resp.StatusCode, string(resp.RawBody))
	}
}
