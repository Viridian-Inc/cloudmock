package network

import (
	"net/http"
	"net/url"
	"testing"

	gojson "github.com/goccy/go-json"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
)

func TestVirtualNetworkAndNetworkSecurityGroupLifecycle(t *testing.T) {
	svc := New()

	vnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2025-05-01"
	vnetPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"addressSpace":{"addressPrefixes":["10.10.0.0/16"]},
			"subnets":[{"name":"default","properties":{"addressPrefix":"10.10.0.0/24"}}]
		}
	}`)

	createVNetResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, vnetURL, vnetPayload))
	if err != nil {
		t.Fatalf("create virtual network returned error: %v", err)
	}
	if createVNetResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create virtual network status 201, got %d; body=%s", createVNetResp.StatusCode, string(createVNetResp.RawBody))
	}
	createdVNet := decodeNetworkResponse(t, createVNetResp)
	if createdVNet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a" {
		t.Fatalf("unexpected virtual network id: %v", createdVNet["id"])
	}
	if createdVNet["name"] != "vnet-a" || createdVNet["type"] != "Microsoft.Network/virtualNetworks" || createdVNet["location"] != "eastus" {
		t.Fatalf("unexpected virtual network identity fields: %v", createdVNet)
	}
	vnetProps := createdVNet["properties"].(map[string]any)
	if vnetProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected virtual network state: %v", vnetProps)
	}
	subnets := vnetProps["subnets"].([]any)
	if len(subnets) != 1 {
		t.Fatalf("expected one subnet, got %d in %v", len(subnets), subnets)
	}
	subnet := subnets[0].(map[string]any)
	if subnet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/default" {
		t.Fatalf("unexpected subnet id: %v", subnet["id"])
	}

	getVNetResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, vnetURL, nil))
	if err != nil {
		t.Fatalf("get virtual network returned error: %v", err)
	}
	if getVNetResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get virtual network status 200, got %d; body=%s", getVNetResp.StatusCode, string(getVNetResp.RawBody))
	}

	listVNetResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list virtual networks returned error: %v", err)
	}
	listedVNets := decodeNetworkResponse(t, listVNetResp)
	vnetValues := listedVNets["value"].([]any)
	if len(vnetValues) != 1 {
		t.Fatalf("expected one virtual network in list, got %d in %v", len(vnetValues), listedVNets)
	}

	nsgURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a?api-version=2025-05-01"
	nsgPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"securityRules":[{"name":"allow-http","properties":{"priority":100,"direction":"Inbound","access":"Allow","protocol":"Tcp","sourceAddressPrefix":"*","sourcePortRange":"*","destinationAddressPrefix":"*","destinationPortRange":"80"}}]
		}
	}`)

	createNSGResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, nsgURL, nsgPayload))
	if err != nil {
		t.Fatalf("create network security group returned error: %v", err)
	}
	if createNSGResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create network security group status 201, got %d; body=%s", createNSGResp.StatusCode, string(createNSGResp.RawBody))
	}
	createdNSG := decodeNetworkResponse(t, createNSGResp)
	if createdNSG["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a" {
		t.Fatalf("unexpected network security group id: %v", createdNSG["id"])
	}
	if createdNSG["name"] != "nsg-a" || createdNSG["type"] != "Microsoft.Network/networkSecurityGroups" || createdNSG["location"] != "eastus" {
		t.Fatalf("unexpected network security group identity fields: %v", createdNSG)
	}
	nsgProps := createdNSG["properties"].(map[string]any)
	if nsgProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected network security group state: %v", nsgProps)
	}
	rules := nsgProps["securityRules"].([]any)
	if len(rules) != 1 {
		t.Fatalf("expected one security rule, got %d in %v", len(rules), rules)
	}
	rule := rules[0].(map[string]any)
	if rule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a/securityRules/allow-http" {
		t.Fatalf("unexpected security rule id: %v", rule["id"])
	}

	getNSGResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, nsgURL, nil))
	if err != nil {
		t.Fatalf("get network security group returned error: %v", err)
	}
	if getNSGResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get network security group status 200, got %d; body=%s", getNSGResp.StatusCode, string(getNSGResp.RawBody))
	}

	listNSGResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list network security groups returned error: %v", err)
	}
	listedNSGs := decodeNetworkResponse(t, listNSGResp)
	nsgValues := listedNSGs["value"].([]any)
	if len(nsgValues) != 1 {
		t.Fatalf("expected one network security group in list, got %d in %v", len(nsgValues), listedNSGs)
	}

	deleteNSGResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, nsgURL, nil))
	if err != nil {
		t.Fatalf("delete network security group returned error: %v", err)
	}
	if deleteNSGResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete network security group status 202, got %d; body=%s", deleteNSGResp.StatusCode, string(deleteNSGResp.RawBody))
	}

	deleteVNetResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, vnetURL, nil))
	if err != nil {
		t.Fatalf("delete virtual network returned error: %v", err)
	}
	if deleteVNetResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete virtual network status 202, got %d; body=%s", deleteVNetResp.StatusCode, string(deleteVNetResp.RawBody))
	}
}

func TestSubnetLifecycle(t *testing.T) {
	svc := New()

	vnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2025-05-01"
	vnetPayload := []byte(`{
		"location":"eastus",
		"properties":{"addressSpace":{"addressPrefixes":["10.20.0.0/16"]}}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, vnetURL, vnetPayload)); err != nil {
		t.Fatalf("create virtual network returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create virtual network status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	subnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a?api-version=2025-05-01"
	subnetPayload := []byte(`{
		"properties":{
			"addressPrefix":"10.20.1.0/24",
			"serviceEndpoints":[{"service":"Microsoft.Storage","locations":["eastus"]}]
		}
	}`)
	createSubnetResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, subnetURL, subnetPayload))
	if err != nil {
		t.Fatalf("create subnet returned error: %v", err)
	}
	if createSubnetResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subnet status 201, got %d; body=%s", createSubnetResp.StatusCode, string(createSubnetResp.RawBody))
	}
	createdSubnet := decodeNetworkResponse(t, createSubnetResp)
	if createdSubnet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a" {
		t.Fatalf("unexpected subnet id: %v", createdSubnet["id"])
	}
	if createdSubnet["name"] != "subnet-a" || createdSubnet["type"] != "Microsoft.Network/virtualNetworks/subnets" {
		t.Fatalf("unexpected subnet identity fields: %v", createdSubnet)
	}
	subnetProps := createdSubnet["properties"].(map[string]any)
	if subnetProps["provisioningState"] != "Succeeded" || subnetProps["addressPrefix"] != "10.20.1.0/24" {
		t.Fatalf("unexpected subnet properties: %v", subnetProps)
	}

	listSubnetsResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list subnets returned error: %v", err)
	}
	listedSubnets := decodeNetworkResponse(t, listSubnetsResp)
	subnetValues := listedSubnets["value"].([]any)
	if len(subnetValues) != 1 {
		t.Fatalf("expected one subnet in list, got %d in %v", len(subnetValues), listedSubnets)
	}

	getVNetResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, vnetURL, nil))
	if err != nil {
		t.Fatalf("get virtual network returned error: %v", err)
	}
	gotVNet := decodeNetworkResponse(t, getVNetResp)
	vnetSubnets := gotVNet["properties"].(map[string]any)["subnets"].([]any)
	if len(vnetSubnets) != 1 {
		t.Fatalf("expected parent virtual network to include one subnet, got %d in %v", len(vnetSubnets), gotVNet)
	}

	deleteSubnetResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, subnetURL, nil))
	if err != nil {
		t.Fatalf("delete subnet returned error: %v", err)
	}
	if deleteSubnetResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete subnet status 202, got %d; body=%s", deleteSubnetResp.StatusCode, string(deleteSubnetResp.RawBody))
	}
}

func TestSecurityRuleLifecycle(t *testing.T) {
	svc := New()

	nsgURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a?api-version=2025-05-01"
	nsgPayload := []byte(`{"location":"eastus","properties":{}}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, nsgURL, nsgPayload)); err != nil {
		t.Fatalf("create network security group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create network security group status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	ruleURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a/securityRules/allow-http?api-version=2025-05-01"
	rulePayload := []byte(`{
		"properties":{
			"priority":100,
			"direction":"Inbound",
			"access":"Allow",
			"protocol":"Tcp",
			"sourceAddressPrefix":"*",
			"sourcePortRange":"*",
			"destinationAddressPrefix":"*",
			"destinationPortRange":"80"
		}
	}`)
	createRuleResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, ruleURL, rulePayload))
	if err != nil {
		t.Fatalf("create security rule returned error: %v", err)
	}
	if createRuleResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create security rule status 201, got %d; body=%s", createRuleResp.StatusCode, string(createRuleResp.RawBody))
	}
	createdRule := decodeNetworkResponse(t, createRuleResp)
	if createdRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a/securityRules/allow-http" {
		t.Fatalf("unexpected security rule id: %v", createdRule["id"])
	}
	if createdRule["name"] != "allow-http" || createdRule["type"] != "Microsoft.Network/networkSecurityGroups/securityRules" {
		t.Fatalf("unexpected security rule identity fields: %v", createdRule)
	}
	ruleProps := createdRule["properties"].(map[string]any)
	if ruleProps["provisioningState"] != "Succeeded" || ruleProps["priority"].(float64) != 100 {
		t.Fatalf("unexpected security rule properties: %v", ruleProps)
	}

	listRulesResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a/securityRules?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list security rules returned error: %v", err)
	}
	listedRules := decodeNetworkResponse(t, listRulesResp)
	ruleValues := listedRules["value"].([]any)
	if len(ruleValues) != 1 {
		t.Fatalf("expected one security rule in list, got %d in %v", len(ruleValues), listedRules)
	}

	getNSGResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, nsgURL, nil))
	if err != nil {
		t.Fatalf("get network security group returned error: %v", err)
	}
	gotNSG := decodeNetworkResponse(t, getNSGResp)
	securityRules := gotNSG["properties"].(map[string]any)["securityRules"].([]any)
	if len(securityRules) != 1 {
		t.Fatalf("expected parent network security group to include one security rule, got %d in %v", len(securityRules), gotNSG)
	}

	deleteRuleResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, ruleURL, nil))
	if err != nil {
		t.Fatalf("delete security rule returned error: %v", err)
	}
	if deleteRuleResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete security rule status 202, got %d; body=%s", deleteRuleResp.StatusCode, string(deleteRuleResp.RawBody))
	}
}

func TestPublicIPAddressAndNetworkInterfaceLifecycle(t *testing.T) {
	svc := New()

	vnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2025-05-01"
	vnetPayload := []byte(`{
		"location":"eastus",
		"properties":{"addressSpace":{"addressPrefixes":["10.40.0.0/16"]}}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, vnetURL, vnetPayload)); err != nil {
		t.Fatalf("create virtual network returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create virtual network status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	subnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a?api-version=2025-05-01"
	subnetPayload := []byte(`{"properties":{"addressPrefix":"10.40.1.0/24"}}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, subnetURL, subnetPayload)); err != nil {
		t.Fatalf("create subnet returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subnet status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	nsgURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a?api-version=2025-05-01"
	nsgPayload := []byte(`{"location":"eastus","properties":{}}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, nsgURL, nsgPayload)); err != nil {
		t.Fatalf("create network security group returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create network security group status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	publicIPURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a?api-version=2025-05-01"
	publicIPPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"tags":{"env":"test"},
		"properties":{
			"publicIPAllocationMethod":"Static",
			"publicIPAddressVersion":"IPv4",
			"dnsSettings":{"domainNameLabel":"cloudmock-pip"}
		},
		"zones":["1"]
	}`)
	createPublicIPResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, publicIPURL, publicIPPayload))
	if err != nil {
		t.Fatalf("create public IP returned error: %v", err)
	}
	if createPublicIPResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create public IP status 201, got %d; body=%s", createPublicIPResp.StatusCode, string(createPublicIPResp.RawBody))
	}
	publicIP := decodeNetworkResponse(t, createPublicIPResp)
	if publicIP["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a" {
		t.Fatalf("unexpected public IP id: %v", publicIP["id"])
	}
	if publicIP["name"] != "pip-a" || publicIP["type"] != "Microsoft.Network/publicIPAddresses" || publicIP["location"] != "eastus" {
		t.Fatalf("unexpected public IP identity fields: %v", publicIP)
	}
	publicIPProps := publicIP["properties"].(map[string]any)
	if publicIPProps["provisioningState"] != "Succeeded" || publicIPProps["ipAddress"] != "203.0.113.10" {
		t.Fatalf("unexpected public IP properties: %v", publicIPProps)
	}
	dnsSettings := publicIPProps["dnsSettings"].(map[string]any)
	if dnsSettings["fqdn"] != "cloudmock-pip.eastus.cloudapp.azure.com" {
		t.Fatalf("unexpected public IP DNS settings: %v", dnsSettings)
	}

	listPublicIPsResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list public IPs returned error: %v", err)
	}
	listedPublicIPs := decodeNetworkResponse(t, listPublicIPsResp)
	if len(listedPublicIPs["value"].([]any)) != 1 {
		t.Fatalf("expected one public IP in list, got %v", listedPublicIPs)
	}

	nicURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a?api-version=2025-05-01"
	nicPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"enableAcceleratedNetworking":false,
			"networkSecurityGroup":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a"},
			"ipConfigurations":[{
				"name":"ipconfig1",
				"properties":{
					"primary":true,
					"privateIPAllocationMethod":"Dynamic",
					"subnet":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a"},
					"publicIPAddress":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a"}
				}
			}]
		}
	}`)
	createNICResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, nicURL, nicPayload))
	if err != nil {
		t.Fatalf("create network interface returned error: %v", err)
	}
	if createNICResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create network interface status 201, got %d; body=%s", createNICResp.StatusCode, string(createNICResp.RawBody))
	}
	nic := decodeNetworkResponse(t, createNICResp)
	if nic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a" {
		t.Fatalf("unexpected network interface id: %v", nic["id"])
	}
	if nic["name"] != "nic-a" || nic["type"] != "Microsoft.Network/networkInterfaces" || nic["location"] != "eastus" {
		t.Fatalf("unexpected network interface identity fields: %v", nic)
	}
	nicProps := nic["properties"].(map[string]any)
	if nicProps["provisioningState"] != "Succeeded" || nicProps["macAddress"] != "00-0D-3A-00-00-01" {
		t.Fatalf("unexpected network interface properties: %v", nicProps)
	}
	ipConfigurations := nicProps["ipConfigurations"].([]any)
	if len(ipConfigurations) != 1 {
		t.Fatalf("expected one NIC IP configuration, got %v", ipConfigurations)
	}
	ipConfig := ipConfigurations[0].(map[string]any)
	if ipConfig["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a/ipConfigurations/ipconfig1" {
		t.Fatalf("unexpected NIC IP configuration id: %v", ipConfig["id"])
	}

	listNICsResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list network interfaces returned error: %v", err)
	}
	listedNICs := decodeNetworkResponse(t, listNICsResp)
	if len(listedNICs["value"].([]any)) != 1 {
		t.Fatalf("expected one network interface in list, got %v", listedNICs)
	}

	deleteNICResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, nicURL, nil))
	if err != nil {
		t.Fatalf("delete network interface returned error: %v", err)
	}
	if deleteNICResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete network interface status 202, got %d; body=%s", deleteNICResp.StatusCode, string(deleteNICResp.RawBody))
	}

	deletePublicIPResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, publicIPURL, nil))
	if err != nil {
		t.Fatalf("delete public IP returned error: %v", err)
	}
	if deletePublicIPResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete public IP status 202, got %d; body=%s", deletePublicIPResp.StatusCode, string(deletePublicIPResp.RawBody))
	}
}

func TestLoadBalancerLifecycle(t *testing.T) {
	svc := New()

	publicIPURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a?api-version=2025-05-01"
	publicIPPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"properties":{"publicIPAllocationMethod":"Static"}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, publicIPURL, publicIPPayload)); err != nil {
		t.Fatalf("create public IP returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create public IP status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	loadBalancerURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a?api-version=2025-05-01"
	loadBalancerPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"tags":{"env":"test"},
		"properties":{
			"frontendIPConfigurations":[{
				"name":"frontend",
				"properties":{
					"publicIPAddress":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a"}
				}
			}],
			"backendAddressPools":[{"name":"backend","properties":{}}],
			"probes":[{"name":"http-probe","properties":{"protocol":"Tcp","port":80,"intervalInSeconds":5,"numberOfProbes":2}}],
			"loadBalancingRules":[{"name":"http-rule","properties":{"protocol":"Tcp","frontendPort":80,"backendPort":80}}],
			"inboundNatRules":[{"name":"ssh-rule","properties":{"protocol":"Tcp","frontendPort":22,"backendPort":22}}],
			"outboundRules":[{"name":"egress","properties":{"protocol":"All"}}]
		}
	}`)
	createLoadBalancerResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, loadBalancerURL, loadBalancerPayload))
	if err != nil {
		t.Fatalf("create load balancer returned error: %v", err)
	}
	if createLoadBalancerResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create load balancer status 201, got %d; body=%s", createLoadBalancerResp.StatusCode, string(createLoadBalancerResp.RawBody))
	}
	loadBalancer := decodeNetworkResponse(t, createLoadBalancerResp)
	if loadBalancer["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a" {
		t.Fatalf("unexpected load balancer id: %v", loadBalancer["id"])
	}
	if loadBalancer["name"] != "lb-a" || loadBalancer["type"] != "Microsoft.Network/loadBalancers" || loadBalancer["location"] != "eastus" {
		t.Fatalf("unexpected load balancer identity fields: %v", loadBalancer)
	}
	lbProps := loadBalancer["properties"].(map[string]any)
	if lbProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected load balancer state: %v", lbProps)
	}
	frontend := lbProps["frontendIPConfigurations"].([]any)[0].(map[string]any)
	if frontend["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/frontendIPConfigurations/frontend" {
		t.Fatalf("unexpected frontend IP configuration id: %v", frontend["id"])
	}
	backend := lbProps["backendAddressPools"].([]any)[0].(map[string]any)
	if backend["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/backendAddressPools/backend" {
		t.Fatalf("unexpected backend pool id: %v", backend["id"])
	}
	probe := lbProps["probes"].([]any)[0].(map[string]any)
	if probe["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/probes/http-probe" {
		t.Fatalf("unexpected probe id: %v", probe["id"])
	}
	rule := lbProps["loadBalancingRules"].([]any)[0].(map[string]any)
	if rule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/loadBalancingRules/http-rule" {
		t.Fatalf("unexpected load balancing rule id: %v", rule["id"])
	}
	inboundRule := lbProps["inboundNatRules"].([]any)[0].(map[string]any)
	if inboundRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/inboundNatRules/ssh-rule" {
		t.Fatalf("unexpected inbound NAT rule id: %v", inboundRule["id"])
	}
	outboundRule := lbProps["outboundRules"].([]any)[0].(map[string]any)
	if outboundRule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a/outboundRules/egress" {
		t.Fatalf("unexpected outbound rule id: %v", outboundRule["id"])
	}

	getLoadBalancerResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, loadBalancerURL, nil))
	if err != nil {
		t.Fatalf("get load balancer returned error: %v", err)
	}
	if getLoadBalancerResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get load balancer status 200, got %d; body=%s", getLoadBalancerResp.StatusCode, string(getLoadBalancerResp.RawBody))
	}

	listLoadBalancersResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list load balancers returned error: %v", err)
	}
	listedLoadBalancers := decodeNetworkResponse(t, listLoadBalancersResp)
	if len(listedLoadBalancers["value"].([]any)) != 1 {
		t.Fatalf("expected one load balancer in list, got %v", listedLoadBalancers)
	}

	deleteLoadBalancerResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, loadBalancerURL, nil))
	if err != nil {
		t.Fatalf("delete load balancer returned error: %v", err)
	}
	if deleteLoadBalancerResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete load balancer status 202, got %d; body=%s", deleteLoadBalancerResp.StatusCode, string(deleteLoadBalancerResp.RawBody))
	}
}

func TestApplicationGatewayLifecycle(t *testing.T) {
	svc := New()

	vnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2025-05-01"
	vnetPayload := []byte(`{
		"location":"eastus",
		"properties":{"addressSpace":{"addressPrefixes":["10.50.0.0/16"]}}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, vnetURL, vnetPayload)); err != nil {
		t.Fatalf("create virtual network returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create virtual network status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	subnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/appgw-subnet?api-version=2025-05-01"
	subnetPayload := []byte(`{"properties":{"addressPrefix":"10.50.1.0/24"}}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, subnetURL, subnetPayload)); err != nil {
		t.Fatalf("create subnet returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subnet status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	publicIPURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-appgw?api-version=2025-05-01"
	publicIPPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard","tier":"Regional"},
		"properties":{"publicIPAllocationMethod":"Static"}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, publicIPURL, publicIPPayload)); err != nil {
		t.Fatalf("create public IP returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create public IP status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	appGatewayURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a?api-version=2025-05-01"
	appGatewayPayload := []byte(`{
		"location":"eastus",
		"sku":{"name":"Standard_v2","tier":"Standard_v2","capacity":2},
		"tags":{"env":"test"},
		"properties":{
			"gatewayIPConfigurations":[{
				"name":"appgw-ipconfig",
				"properties":{"subnet":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/appgw-subnet"}}
			}],
			"frontendIPConfigurations":[{
				"name":"frontend",
				"properties":{"publicIPAddress":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-appgw"}}
			}],
			"frontendPorts":[{"name":"http-port","properties":{"port":80}}],
			"backendAddressPools":[{"name":"backend","properties":{"backendAddresses":[{"fqdn":"app.internal.test"}]}}],
			"backendHttpSettingsCollection":[{"name":"backend-http","properties":{"port":80,"protocol":"Http","cookieBasedAffinity":"Disabled"}}],
			"httpListeners":[{"name":"listener","properties":{"protocol":"Http"}}],
			"requestRoutingRules":[{"name":"rule","properties":{"ruleType":"Basic","priority":100}}],
			"probes":[{"name":"probe","properties":{"protocol":"Http","path":"/healthz","interval":30}}],
			"redirectConfigurations":[{"name":"redirect","properties":{"redirectType":"Permanent"}}],
			"sslCertificates":[{"name":"ssl-cert","properties":{"data":"ZmFrZQ=="}}],
			"trustedRootCertificates":[{"name":"root-cert","properties":{"data":"ZmFrZQ=="}}],
			"urlPathMaps":[{"name":"paths","properties":{"defaultBackendAddressPool":{"id":"backend"}}}]
		}
	}`)
	createAppGatewayResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, appGatewayURL, appGatewayPayload))
	if err != nil {
		t.Fatalf("create application gateway returned error: %v", err)
	}
	if createAppGatewayResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create application gateway status 201, got %d; body=%s", createAppGatewayResp.StatusCode, string(createAppGatewayResp.RawBody))
	}
	appGateway := decodeNetworkResponse(t, createAppGatewayResp)
	if appGateway["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a" {
		t.Fatalf("unexpected application gateway id: %v", appGateway["id"])
	}
	if appGateway["name"] != "appgw-a" || appGateway["type"] != "Microsoft.Network/applicationGateways" || appGateway["location"] != "eastus" {
		t.Fatalf("unexpected application gateway identity fields: %v", appGateway)
	}
	appGatewayProps := appGateway["properties"].(map[string]any)
	if appGatewayProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected application gateway state: %v", appGatewayProps)
	}
	expectedChildIDs := map[string]string{
		"gatewayIPConfigurations":       "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/gatewayIPConfigurations/appgw-ipconfig",
		"frontendIPConfigurations":      "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/frontendIPConfigurations/frontend",
		"frontendPorts":                 "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/frontendPorts/http-port",
		"backendAddressPools":           "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/backendAddressPools/backend",
		"backendHttpSettingsCollection": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/backendHttpSettingsCollection/backend-http",
		"httpListeners":                 "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/httpListeners/listener",
		"requestRoutingRules":           "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/requestRoutingRules/rule",
		"probes":                        "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/probes/probe",
		"redirectConfigurations":        "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/redirectConfigurations/redirect",
		"sslCertificates":               "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/sslCertificates/ssl-cert",
		"trustedRootCertificates":       "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/trustedRootCertificates/root-cert",
		"urlPathMaps":                   "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a/urlPathMaps/paths",
	}
	for collection, expectedID := range expectedChildIDs {
		children := appGatewayProps[collection].([]any)
		if len(children) != 1 {
			t.Fatalf("expected one %s child, got %v", collection, children)
		}
		child := children[0].(map[string]any)
		if child["id"] != expectedID {
			t.Fatalf("unexpected %s child id: %v", collection, child["id"])
		}
	}

	getAppGatewayResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, appGatewayURL, nil))
	if err != nil {
		t.Fatalf("get application gateway returned error: %v", err)
	}
	if getAppGatewayResp.StatusCode != http.StatusOK {
		t.Fatalf("expected get application gateway status 200, got %d; body=%s", getAppGatewayResp.StatusCode, string(getAppGatewayResp.RawBody))
	}

	listAppGatewaysResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list application gateways returned error: %v", err)
	}
	listedAppGateways := decodeNetworkResponse(t, listAppGatewaysResp)
	if len(listedAppGateways["value"].([]any)) != 1 {
		t.Fatalf("expected one application gateway in list, got %v", listedAppGateways)
	}

	deleteAppGatewayResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, appGatewayURL, nil))
	if err != nil {
		t.Fatalf("delete application gateway returned error: %v", err)
	}
	if deleteAppGatewayResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete application gateway status 202, got %d; body=%s", deleteAppGatewayResp.StatusCode, string(deleteAppGatewayResp.RawBody))
	}
}

func TestPrivateEndpointAndPrivateDNSZoneGroupLifecycle(t *testing.T) {
	svc := New()

	vnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a?api-version=2025-05-01"
	vnetPayload := []byte(`{
		"location":"eastus",
		"properties":{"addressSpace":{"addressPrefixes":["10.60.0.0/16"]}}
	}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, vnetURL, vnetPayload)); err != nil {
		t.Fatalf("create virtual network returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create virtual network status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	subnetURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/private-endpoints?api-version=2025-05-01"
	subnetPayload := []byte(`{"properties":{"addressPrefix":"10.60.1.0/24","privateEndpointNetworkPolicies":"Disabled"}}`)
	if resp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, subnetURL, subnetPayload)); err != nil {
		t.Fatalf("create subnet returned error: %v", err)
	} else if resp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create subnet status 201, got %d; body=%s", resp.StatusCode, string(resp.RawBody))
	}

	privateEndpointURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a?api-version=2025-05-01"
	privateEndpointPayload := []byte(`{
		"location":"eastus",
		"tags":{"env":"test"},
		"properties":{
			"subnet":{"id":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/private-endpoints"},
			"privateLinkServiceConnections":[{
				"name":"blob",
				"properties":{
					"privateLinkServiceId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/stgacct",
					"groupIds":["blob"],
					"requestMessage":"Please approve"
				}
			}],
			"ipConfigurations":[{
				"name":"primary",
				"properties":{
					"groupId":"blob",
					"memberName":"blob",
					"privateIPAddress":"10.60.1.4"
				}
			}]
		}
	}`)
	createPrivateEndpointResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, privateEndpointURL, privateEndpointPayload))
	if err != nil {
		t.Fatalf("create private endpoint returned error: %v", err)
	}
	if createPrivateEndpointResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create private endpoint status 201, got %d; body=%s", createPrivateEndpointResp.StatusCode, string(createPrivateEndpointResp.RawBody))
	}
	privateEndpoint := decodeNetworkResponse(t, createPrivateEndpointResp)
	if privateEndpoint["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a" {
		t.Fatalf("unexpected private endpoint id: %v", privateEndpoint["id"])
	}
	if privateEndpoint["name"] != "pe-a" || privateEndpoint["type"] != "Microsoft.Network/privateEndpoints" || privateEndpoint["location"] != "eastus" {
		t.Fatalf("unexpected private endpoint identity fields: %v", privateEndpoint)
	}
	peProps := privateEndpoint["properties"].(map[string]any)
	if peProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected private endpoint state: %v", peProps)
	}
	connections := peProps["privateLinkServiceConnections"].([]any)
	if len(connections) != 1 {
		t.Fatalf("expected one private link service connection, got %v", connections)
	}
	connection := connections[0].(map[string]any)
	if connection["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateLinkServiceConnections/blob" {
		t.Fatalf("unexpected private link service connection id: %v", connection["id"])
	}
	ipConfigurations := peProps["ipConfigurations"].([]any)
	if len(ipConfigurations) != 1 {
		t.Fatalf("expected one private endpoint IP configuration, got %v", ipConfigurations)
	}
	ipConfiguration := ipConfigurations[0].(map[string]any)
	if ipConfiguration["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/ipConfigurations/primary" {
		t.Fatalf("unexpected private endpoint IP configuration id: %v", ipConfiguration["id"])
	}

	zoneGroupURL := "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateDnsZoneGroups/default?api-version=2025-05-01"
	zoneGroupPayload := []byte(`{
		"properties":{
			"privateDnsZoneConfigs":[{
				"name":"blob-zone",
				"properties":{
					"privateDnsZoneId":"/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateDnsZones/privatelink.blob.core.windows.net"
				}
			}]
		}
	}`)
	createZoneGroupResp, err := svc.HandleRequest(networkCtx(t, http.MethodPut, zoneGroupURL, zoneGroupPayload))
	if err != nil {
		t.Fatalf("create private DNS zone group returned error: %v", err)
	}
	if createZoneGroupResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected create private DNS zone group status 201, got %d; body=%s", createZoneGroupResp.StatusCode, string(createZoneGroupResp.RawBody))
	}
	zoneGroup := decodeNetworkResponse(t, createZoneGroupResp)
	if zoneGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateDnsZoneGroups/default" {
		t.Fatalf("unexpected private DNS zone group id: %v", zoneGroup["id"])
	}
	if zoneGroup["name"] != "default" || zoneGroup["type"] != "Microsoft.Network/privateEndpoints/privateDnsZoneGroups" {
		t.Fatalf("unexpected private DNS zone group identity fields: %v", zoneGroup)
	}
	zoneGroupProps := zoneGroup["properties"].(map[string]any)
	if zoneGroupProps["provisioningState"] != "Succeeded" {
		t.Fatalf("unexpected private DNS zone group state: %v", zoneGroupProps)
	}
	zoneConfigs := zoneGroupProps["privateDnsZoneConfigs"].([]any)
	if len(zoneConfigs) != 1 {
		t.Fatalf("expected one private DNS zone config, got %v", zoneConfigs)
	}
	zoneConfig := zoneConfigs[0].(map[string]any)
	if zoneConfig["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateDnsZoneGroups/default/privateDnsZoneConfigs/blob-zone" {
		t.Fatalf("unexpected private DNS zone config id: %v", zoneConfig["id"])
	}

	getPrivateEndpointResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, privateEndpointURL, nil))
	if err != nil {
		t.Fatalf("get private endpoint returned error: %v", err)
	}
	gotPrivateEndpoint := decodeNetworkResponse(t, getPrivateEndpointResp)
	parentZoneGroups := gotPrivateEndpoint["properties"].(map[string]any)["privateDnsZoneGroups"].([]any)
	if len(parentZoneGroups) != 1 {
		t.Fatalf("expected parent private endpoint to include one private DNS zone group, got %v", gotPrivateEndpoint)
	}

	listPrivateEndpointsResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list private endpoints returned error: %v", err)
	}
	listedPrivateEndpoints := decodeNetworkResponse(t, listPrivateEndpointsResp)
	if len(listedPrivateEndpoints["value"].([]any)) != 1 {
		t.Fatalf("expected one private endpoint in list, got %v", listedPrivateEndpoints)
	}

	listZoneGroupsResp, err := svc.HandleRequest(networkCtx(t, http.MethodGet, "https://management.azure.com/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateDnsZoneGroups?api-version=2025-05-01", nil))
	if err != nil {
		t.Fatalf("list private DNS zone groups returned error: %v", err)
	}
	listedZoneGroups := decodeNetworkResponse(t, listZoneGroupsResp)
	if len(listedZoneGroups["value"].([]any)) != 1 {
		t.Fatalf("expected one private DNS zone group in list, got %v", listedZoneGroups)
	}

	deleteZoneGroupResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, zoneGroupURL, nil))
	if err != nil {
		t.Fatalf("delete private DNS zone group returned error: %v", err)
	}
	if deleteZoneGroupResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete private DNS zone group status 202, got %d; body=%s", deleteZoneGroupResp.StatusCode, string(deleteZoneGroupResp.RawBody))
	}

	deletePrivateEndpointResp, err := svc.HandleRequest(networkCtx(t, http.MethodDelete, privateEndpointURL, nil))
	if err != nil {
		t.Fatalf("delete private endpoint returned error: %v", err)
	}
	if deletePrivateEndpointResp.StatusCode != http.StatusAccepted {
		t.Fatalf("expected delete private endpoint status 202, got %d; body=%s", deletePrivateEndpointResp.StatusCode, string(deletePrivateEndpointResp.RawBody))
	}
}

func TestNetworkChildTemplateProvisioning(t *testing.T) {
	svc := New()

	vnetResource := map[string]any{
		"type":     "Microsoft.Network/virtualNetworks",
		"name":     "vnet-a",
		"location": "eastus",
		"properties": map[string]any{
			"addressSpace": map[string]any{"addressPrefixes": []any{"10.30.0.0/16"}},
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", vnetResource); err != nil {
		t.Fatalf("provision virtual network returned error: %v", err)
	}
	subnetResource := map[string]any{
		"type": "Microsoft.Network/virtualNetworks/subnets",
		"name": "vnet-a/subnet-a",
		"properties": map[string]any{
			"addressPrefix": "10.30.1.0/24",
		},
	}
	subnetResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", subnetResource)
	if err != nil {
		t.Fatalf("provision subnet returned error: %v", err)
	}
	subnet := subnetResult.(map[string]any)
	if subnet["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a" {
		t.Fatalf("unexpected provisioned subnet id: %v", subnet["id"])
	}
	if subnet["type"] != "Microsoft.Network/virtualNetworks/subnets" {
		t.Fatalf("unexpected provisioned subnet type: %v", subnet["type"])
	}

	nsgResource := map[string]any{
		"type":     "Microsoft.Network/networkSecurityGroups",
		"name":     "nsg-a",
		"location": "eastus",
		"properties": map[string]any{
			"securityRules": []any{},
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", nsgResource); err != nil {
		t.Fatalf("provision network security group returned error: %v", err)
	}
	ruleResource := map[string]any{
		"type": "Microsoft.Network/networkSecurityGroups/securityRules",
		"name": "nsg-a/allow-http",
		"properties": map[string]any{
			"priority":                 100,
			"direction":                "Inbound",
			"access":                   "Allow",
			"protocol":                 "Tcp",
			"sourceAddressPrefix":      "*",
			"sourcePortRange":          "*",
			"destinationAddressPrefix": "*",
			"destinationPortRange":     "80",
		},
	}
	ruleResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", ruleResource)
	if err != nil {
		t.Fatalf("provision security rule returned error: %v", err)
	}
	rule := ruleResult.(map[string]any)
	if rule["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a/securityRules/allow-http" {
		t.Fatalf("unexpected provisioned security rule id: %v", rule["id"])
	}
	if rule["type"] != "Microsoft.Network/networkSecurityGroups/securityRules" {
		t.Fatalf("unexpected provisioned security rule type: %v", rule["type"])
	}

	publicIPResource := map[string]any{
		"type":     "Microsoft.Network/publicIPAddresses",
		"name":     "pip-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Standard", "tier": "Regional"},
		"properties": map[string]any{
			"publicIPAllocationMethod": "Static",
			"dnsSettings": map[string]any{
				"domainNameLabel": "cloudmock-pip",
			},
		},
	}
	publicIPResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", publicIPResource)
	if err != nil {
		t.Fatalf("provision public IP returned error: %v", err)
	}
	publicIP := publicIPResult.(map[string]any)
	if publicIP["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a" {
		t.Fatalf("unexpected provisioned public IP id: %v", publicIP["id"])
	}
	if publicIP["type"] != "Microsoft.Network/publicIPAddresses" {
		t.Fatalf("unexpected provisioned public IP type: %v", publicIP["type"])
	}

	nicResource := map[string]any{
		"type":     "Microsoft.Network/networkInterfaces",
		"name":     "nic-a",
		"location": "eastus",
		"properties": map[string]any{
			"networkSecurityGroup": map[string]any{"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkSecurityGroups/nsg-a"},
			"ipConfigurations": []any{
				map[string]any{
					"name": "ipconfig1",
					"properties": map[string]any{
						"subnet":          map[string]any{"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a"},
						"publicIPAddress": map[string]any{"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a"},
					},
				},
			},
		},
	}
	nicResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", nicResource)
	if err != nil {
		t.Fatalf("provision network interface returned error: %v", err)
	}
	nic := nicResult.(map[string]any)
	if nic["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/networkInterfaces/nic-a" {
		t.Fatalf("unexpected provisioned NIC id: %v", nic["id"])
	}
	if nic["type"] != "Microsoft.Network/networkInterfaces" {
		t.Fatalf("unexpected provisioned NIC type: %v", nic["type"])
	}

	loadBalancerResource := map[string]any{
		"type":     "Microsoft.Network/loadBalancers",
		"name":     "lb-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Standard", "tier": "Regional"},
		"properties": map[string]any{
			"frontendIPConfigurations": []any{
				map[string]any{
					"name": "frontend",
					"properties": map[string]any{
						"publicIPAddress": map[string]any{"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/publicIPAddresses/pip-a"},
					},
				},
			},
			"backendAddressPools": []any{
				map[string]any{"name": "backend", "properties": map[string]any{}},
			},
			"loadBalancingRules": []any{
				map[string]any{"name": "http-rule", "properties": map[string]any{"protocol": "Tcp", "frontendPort": 80, "backendPort": 80}},
			},
		},
	}
	loadBalancerResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", loadBalancerResource)
	if err != nil {
		t.Fatalf("provision load balancer returned error: %v", err)
	}
	loadBalancer := loadBalancerResult.(map[string]any)
	if loadBalancer["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/loadBalancers/lb-a" {
		t.Fatalf("unexpected provisioned load balancer id: %v", loadBalancer["id"])
	}
	if loadBalancer["type"] != "Microsoft.Network/loadBalancers" {
		t.Fatalf("unexpected provisioned load balancer type: %v", loadBalancer["type"])
	}

	appGatewayResource := map[string]any{
		"type":     "Microsoft.Network/applicationGateways",
		"name":     "appgw-a",
		"location": "eastus",
		"sku":      map[string]any{"name": "Standard_v2", "tier": "Standard_v2", "capacity": 2},
		"properties": map[string]any{
			"frontendPorts": []any{
				map[string]any{"name": "http-port", "properties": map[string]any{"port": 80}},
			},
			"backendAddressPools": []any{
				map[string]any{"name": "backend", "properties": map[string]any{}},
			},
			"httpListeners": []any{
				map[string]any{"name": "listener", "properties": map[string]any{"protocol": "Http"}},
			},
			"requestRoutingRules": []any{
				map[string]any{"name": "rule", "properties": map[string]any{"ruleType": "Basic"}},
			},
		},
	}
	appGatewayResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", appGatewayResource)
	if err != nil {
		t.Fatalf("provision application gateway returned error: %v", err)
	}
	appGateway := appGatewayResult.(map[string]any)
	if appGateway["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/applicationGateways/appgw-a" {
		t.Fatalf("unexpected provisioned application gateway id: %v", appGateway["id"])
	}
	if appGateway["type"] != "Microsoft.Network/applicationGateways" {
		t.Fatalf("unexpected provisioned application gateway type: %v", appGateway["type"])
	}

	privateEndpointResource := map[string]any{
		"type":     "Microsoft.Network/privateEndpoints",
		"name":     "pe-a",
		"location": "eastus",
		"properties": map[string]any{
			"subnet": map[string]any{"id": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/virtualNetworks/vnet-a/subnets/subnet-a"},
			"privateLinkServiceConnections": []any{
				map[string]any{
					"name": "blob",
					"properties": map[string]any{
						"privateLinkServiceId": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Storage/storageAccounts/stgacct",
						"groupIds":             []any{"blob"},
					},
				},
			},
		},
	}
	if _, err := svc.ProvisionTemplateResource("sub-1", "rg-a", privateEndpointResource); err != nil {
		t.Fatalf("provision private endpoint returned error: %v", err)
	}
	zoneGroupResource := map[string]any{
		"type": "Microsoft.Network/privateEndpoints/privateDnsZoneGroups",
		"name": "pe-a/default",
		"properties": map[string]any{
			"privateDnsZoneConfigs": []any{
				map[string]any{
					"name": "blob-zone",
					"properties": map[string]any{
						"privateDnsZoneId": "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateDnsZones/privatelink.blob.core.windows.net",
					},
				},
			},
		},
	}
	zoneGroupResult, err := svc.ProvisionTemplateResource("sub-1", "rg-a", zoneGroupResource)
	if err != nil {
		t.Fatalf("provision private DNS zone group returned error: %v", err)
	}
	provisionedZoneGroup := zoneGroupResult.(map[string]any)
	if provisionedZoneGroup["id"] != "/subscriptions/sub-1/resourceGroups/rg-a/providers/Microsoft.Network/privateEndpoints/pe-a/privateDnsZoneGroups/default" {
		t.Fatalf("unexpected provisioned private DNS zone group id: %v", provisionedZoneGroup["id"])
	}
	if provisionedZoneGroup["type"] != "Microsoft.Network/privateEndpoints/privateDnsZoneGroups" {
		t.Fatalf("unexpected provisioned private DNS zone group type: %v", provisionedZoneGroup["type"])
	}
}

func TestServiceKeysIncludeTopLevelNetworkResourceTypes(t *testing.T) {
	svc := New()

	seen := make(map[string]bool)
	for _, key := range svc.ServiceKeys() {
		seen[string(key.Provider)+"|"+key.Service+"|"+key.APIVersion] = true
	}

	for _, serviceName := range []string{
		"Microsoft.Network/virtualNetworks",
		"Microsoft.Network/networkSecurityGroups",
		"Microsoft.Network/publicIPAddresses",
		"Microsoft.Network/networkInterfaces",
		"Microsoft.Network/loadBalancers",
		"Microsoft.Network/applicationGateways",
		"Microsoft.Network/privateEndpoints",
	} {
		for _, version := range []string{"2025-05-01", "2023-09-01"} {
			lookup := "azure|" + serviceName + "|" + version
			if !seen[lookup] {
				t.Fatalf("expected service key %s", lookup)
			}
		}
	}
}

func networkCtx(t *testing.T, method, rawURL string, body []byte) *service.RequestContext {
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

func decodeNetworkResponse(t *testing.T, resp *service.Response) map[string]any {
	t.Helper()
	var out map[string]any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		t.Fatalf("decode response: %v; body=%s", err, string(resp.RawBody))
	}
	return out
}
