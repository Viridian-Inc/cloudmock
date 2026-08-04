package network

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

var networkAPIVersions = []string{"2025-05-01", "2023-09-01"}

// NetworkService implements first-slice Azure Network control-plane APIs.
type NetworkService struct {
	mu                   sync.RWMutex
	vnets                map[string]VirtualNetwork
	nsgs                 map[string]NetworkSecurityGroup
	publicIPs            map[string]PublicIPAddress
	networkInterfaces    map[string]NetworkInterface
	loadBalancers        map[string]LoadBalancer
	applicationGateways  map[string]ApplicationGateway
	privateEndpoints     map[string]PrivateEndpoint
	privateDNSZoneGroups map[string]PrivateDNSZoneGroup
	subnets              map[string]Subnet
	securityRules        map[string]SecurityRule
}

func New() *NetworkService {
	return &NetworkService{
		vnets:                make(map[string]VirtualNetwork),
		nsgs:                 make(map[string]NetworkSecurityGroup),
		publicIPs:            make(map[string]PublicIPAddress),
		networkInterfaces:    make(map[string]NetworkInterface),
		loadBalancers:        make(map[string]LoadBalancer),
		applicationGateways:  make(map[string]ApplicationGateway),
		privateEndpoints:     make(map[string]PrivateEndpoint),
		privateDNSZoneGroups: make(map[string]PrivateDNSZoneGroup),
		subnets:              make(map[string]Subnet),
		securityRules:        make(map[string]SecurityRule),
	}
}

func (s *NetworkService) Name() string { return "Microsoft.Network" }

func (s *NetworkService) Actions() []service.Action {
	return []service.Action{
		{Name: "CreateOrUpdateVirtualNetwork", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/virtualNetworks/write"},
		{Name: "GetVirtualNetwork", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/virtualNetworks/read"},
		{Name: "ListVirtualNetworks", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/virtualNetworks/read"},
		{Name: "DeleteVirtualNetwork", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/virtualNetworks/delete"},
		{Name: "CreateOrUpdateSubnet", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/virtualNetworks/subnets/write"},
		{Name: "GetSubnet", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/virtualNetworks/subnets/read"},
		{Name: "ListSubnets", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/virtualNetworks/subnets/read"},
		{Name: "DeleteSubnet", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/virtualNetworks/subnets/delete"},
		{Name: "CreateOrUpdateNetworkSecurityGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/write"},
		{Name: "GetNetworkSecurityGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/read"},
		{Name: "ListNetworkSecurityGroups", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/read"},
		{Name: "DeleteNetworkSecurityGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/delete"},
		{Name: "CreateOrUpdatePublicIPAddress", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/publicIPAddresses/write"},
		{Name: "GetPublicIPAddress", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/publicIPAddresses/read"},
		{Name: "ListPublicIPAddresses", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/publicIPAddresses/read"},
		{Name: "DeletePublicIPAddress", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/publicIPAddresses/delete"},
		{Name: "CreateOrUpdateNetworkInterface", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/networkInterfaces/write"},
		{Name: "GetNetworkInterface", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkInterfaces/read"},
		{Name: "ListNetworkInterfaces", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkInterfaces/read"},
		{Name: "DeleteNetworkInterface", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/networkInterfaces/delete"},
		{Name: "CreateOrUpdateLoadBalancer", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/loadBalancers/write"},
		{Name: "GetLoadBalancer", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/loadBalancers/read"},
		{Name: "ListLoadBalancers", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/loadBalancers/read"},
		{Name: "DeleteLoadBalancer", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/loadBalancers/delete"},
		{Name: "CreateOrUpdateApplicationGateway", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/applicationGateways/write"},
		{Name: "GetApplicationGateway", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/applicationGateways/read"},
		{Name: "ListApplicationGateways", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/applicationGateways/read"},
		{Name: "DeleteApplicationGateway", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/applicationGateways/delete"},
		{Name: "CreateOrUpdatePrivateEndpoint", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/privateEndpoints/write"},
		{Name: "GetPrivateEndpoint", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/privateEndpoints/read"},
		{Name: "ListPrivateEndpoints", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/privateEndpoints/read"},
		{Name: "DeletePrivateEndpoint", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/privateEndpoints/delete"},
		{Name: "CreateOrUpdatePrivateDNSZoneGroup", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/privateEndpoints/privateDnsZoneGroups/write"},
		{Name: "GetPrivateDNSZoneGroup", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/privateEndpoints/privateDnsZoneGroups/read"},
		{Name: "ListPrivateDNSZoneGroups", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/privateEndpoints/privateDnsZoneGroups/read"},
		{Name: "DeletePrivateDNSZoneGroup", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/privateEndpoints/privateDnsZoneGroups/delete"},
		{Name: "CreateOrUpdateSecurityRule", Method: http.MethodPut, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/securityRules/write"},
		{Name: "GetSecurityRule", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/securityRules/read"},
		{Name: "ListSecurityRules", Method: http.MethodGet, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/securityRules/read"},
		{Name: "DeleteSecurityRule", Method: http.MethodDelete, IAMAction: "azure:Microsoft.Network/networkSecurityGroups/securityRules/delete"},
	}
}

func (s *NetworkService) HealthCheck() error { return nil }

func (s *NetworkService) ServiceKeys() []routing.ServiceKey {
	keys := make([]routing.ServiceKey, 0, len(networkAPIVersions)*2)
	for _, version := range networkAPIVersions {
		keys = append(keys,
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/virtualNetworks", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/networkSecurityGroups", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/publicIPAddresses", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/networkInterfaces", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/loadBalancers", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/applicationGateways", APIVersion: version},
			routing.ServiceKey{Provider: routing.ProviderAzure, Service: "Microsoft.Network/privateEndpoints", APIVersion: version},
		)
	}
	return keys
}

func (s *NetworkService) SupportsTemplateResource(resource map[string]any) bool {
	resourceType := stringValue(resource["type"])
	return strings.EqualFold(resourceType, "Microsoft.Network/virtualNetworks") ||
		strings.EqualFold(resourceType, "Microsoft.Network/virtualNetworks/subnets") ||
		strings.EqualFold(resourceType, "Microsoft.Network/networkSecurityGroups") ||
		strings.EqualFold(resourceType, "Microsoft.Network/networkSecurityGroups/securityRules") ||
		strings.EqualFold(resourceType, "Microsoft.Network/publicIPAddresses") ||
		strings.EqualFold(resourceType, "Microsoft.Network/networkInterfaces") ||
		strings.EqualFold(resourceType, "Microsoft.Network/loadBalancers") ||
		strings.EqualFold(resourceType, "Microsoft.Network/applicationGateways") ||
		strings.EqualFold(resourceType, "Microsoft.Network/privateEndpoints") ||
		strings.EqualFold(resourceType, "Microsoft.Network/privateEndpoints/privateDnsZoneGroups")
}

func (s *NetworkService) ProvisionTemplateResource(subscriptionID, resourceGroup string, resource map[string]any) (any, error) {
	if !s.SupportsTemplateResource(resource) {
		return nil, fmt.Errorf("unsupported Network template resource type %q", stringValue(resource["type"]))
	}
	name := stringValue(resource["name"])
	if name == "" {
		return nil, fmt.Errorf("Network template resource is missing name")
	}

	body := map[string]any{
		"location":   resource["location"],
		"sku":        resource["sku"],
		"tags":       resource["tags"],
		"zones":      resource["zones"],
		"properties": resource["properties"],
	}
	data, err := gojson.Marshal(body)
	if err != nil {
		return nil, err
	}

	var resp *service.Response
	resourceType := stringValue(resource["type"])
	switch {
	case strings.EqualFold(resourceType, "Microsoft.Network/virtualNetworks"):
		resp, err = s.createOrUpdateVirtualNetwork(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/virtualNetworks/subnets"):
		virtualNetworkName, subnetName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Network subnet template resource name must be {virtualNetwork}/{subnet}")
		}
		resp, err = s.createOrUpdateSubnet(subscriptionID, resourceGroup, virtualNetworkName, subnetName, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/networkSecurityGroups"):
		resp, err = s.createOrUpdateNetworkSecurityGroup(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/networkSecurityGroups/securityRules"):
		networkSecurityGroupName, ruleName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Network security rule template resource name must be {networkSecurityGroup}/{securityRule}")
		}
		resp, err = s.createOrUpdateSecurityRule(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/publicIPAddresses"):
		resp, err = s.createOrUpdatePublicIPAddress(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/networkInterfaces"):
		resp, err = s.createOrUpdateNetworkInterface(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/loadBalancers"):
		resp, err = s.createOrUpdateLoadBalancer(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/applicationGateways"):
		resp, err = s.createOrUpdateApplicationGateway(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/privateEndpoints"):
		resp, err = s.createOrUpdatePrivateEndpoint(subscriptionID, resourceGroup, name, data)
	case strings.EqualFold(resourceType, "Microsoft.Network/privateEndpoints/privateDnsZoneGroups"):
		privateEndpointName, zoneGroupName, ok := splitNestedName(name)
		if !ok {
			return nil, fmt.Errorf("Network private DNS zone group template resource name must be {privateEndpoint}/{privateDnsZoneGroup}")
		}
		resp, err = s.createOrUpdatePrivateDNSZoneGroup(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName, data)
	default:
		err = fmt.Errorf("unsupported Network template resource type %q", resourceType)
	}
	if err != nil {
		return nil, err
	}
	var out any
	if err := gojson.Unmarshal(resp.RawBody, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *NetworkService) HandleRequest(ctx *service.RequestContext) (*service.Response, error) {
	if ctx == nil || ctx.RawRequest == nil {
		return azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequest", "A raw HTTP request is required.")
	}

	route, ok := parseRoute(ctx.RawRequest.URL.EscapedPath())
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Network route is not implemented.")
	}

	switch {
	case strings.EqualFold(route.ResourceType, "virtualNetworks"):
		if route.ChildType != "" {
			if strings.EqualFold(route.ChildType, "subnets") {
				return s.handleSubnetRequest(ctx, route)
			}
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Network route is not implemented.")
		}
		return s.handleVirtualNetworkRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "networkSecurityGroups"):
		if route.ChildType != "" {
			if strings.EqualFold(route.ChildType, "securityRules") {
				return s.handleSecurityRuleRequest(ctx, route)
			}
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Network route is not implemented.")
		}
		return s.handleNetworkSecurityGroupRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "publicIPAddresses"):
		return s.handlePublicIPAddressRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "networkInterfaces"):
		return s.handleNetworkInterfaceRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "loadBalancers"):
		return s.handleLoadBalancerRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "applicationGateways"):
		return s.handleApplicationGatewayRequest(ctx, route)
	case strings.EqualFold(route.ResourceType, "privateEndpoints"):
		if route.ChildType != "" {
			if strings.EqualFold(route.ChildType, "privateDnsZoneGroups") {
				return s.handlePrivateDNSZoneGroupRequest(ctx, route)
			}
			return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Network route is not implemented.")
		}
		return s.handlePrivateEndpointRequest(ctx, route)
	default:
		return azurearm.ErrorResponse(http.StatusNotFound, "NotFound", "The Network route is not implemented.")
	}
}

func (s *NetworkService) handleVirtualNetworkRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listVirtualNetworks(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateVirtualNetwork(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getVirtualNetwork(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteVirtualNetwork(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleSubnetRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSubnets(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSubnet(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getSubnet(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteSubnet(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleNetworkSecurityGroupRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNetworkSecurityGroups(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateNetworkSecurityGroup(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getNetworkSecurityGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteNetworkSecurityGroup(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleSecurityRuleRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listSecurityRules(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateSecurityRule(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getSecurityRule(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deleteSecurityRule(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handlePublicIPAddressRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listPublicIPAddresses(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdatePublicIPAddress(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getPublicIPAddress(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deletePublicIPAddress(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleNetworkInterfaceRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listNetworkInterfaces(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateNetworkInterface(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getNetworkInterface(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteNetworkInterface(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleLoadBalancerRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listLoadBalancers(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateLoadBalancer(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getLoadBalancer(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteLoadBalancer(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handleApplicationGatewayRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listApplicationGateways(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdateApplicationGateway(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getApplicationGateway(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deleteApplicationGateway(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handlePrivateEndpointRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.Name == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listPrivateEndpoints(route.SubscriptionID, route.ResourceGroup)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdatePrivateEndpoint(route.SubscriptionID, route.ResourceGroup, route.Name, ctx.Body)
	case http.MethodGet:
		return s.getPrivateEndpoint(route.SubscriptionID, route.ResourceGroup, route.Name)
	case http.MethodDelete:
		return s.deletePrivateEndpoint(route.SubscriptionID, route.ResourceGroup, route.Name)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) handlePrivateDNSZoneGroupRequest(ctx *service.RequestContext, route networkRoute) (*service.Response, error) {
	if route.ChildName == "" {
		if ctx.RawRequest.Method == http.MethodGet {
			return s.listPrivateDNSZoneGroups(route.SubscriptionID, route.ResourceGroup, route.Name)
		}
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}

	switch ctx.RawRequest.Method {
	case http.MethodPut:
		return s.createOrUpdatePrivateDNSZoneGroup(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName, ctx.Body)
	case http.MethodGet:
		return s.getPrivateDNSZoneGroup(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	case http.MethodDelete:
		return s.deletePrivateDNSZoneGroup(route.SubscriptionID, route.ResourceGroup, route.Name, route.ChildName)
	default:
		return azurearm.ErrorResponse(http.StatusMethodNotAllowed, "MethodNotAllowed", "The method is not allowed for this route.")
	}
}

func (s *NetworkService) createOrUpdateVirtualNetwork(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
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
	id := virtualNetworkID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	input.Properties["subnets"] = namedChildrenWithIDs(input.Properties["subnets"], id+"/subnets")

	vnet := VirtualNetwork{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/virtualNetworks",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.vnets[key]
	s.vnets[key] = vnet
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, vnet)
}

func (s *NetworkService) getVirtualNetwork(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	vnet, ok := s.vnets[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual network %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, vnet)
}

func (s *NetworkService) listVirtualNetworks(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]VirtualNetwork, 0)
	for key, vnet := range s.vnets {
		if strings.HasPrefix(key, prefix) {
			values = append(values, vnet)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteVirtualNetwork(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.vnets[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual network %q could not be found.", name))
	}
	delete(s.vnets, key)
	childPrefix := key + "/"
	for subnetKey := range s.subnets {
		if strings.HasPrefix(subnetKey, childPrefix) {
			delete(s.subnets, subnetKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateSubnet(subscriptionID, resourceGroup, virtualNetworkName, subnetName string, body []byte) (*service.Response, error) {
	properties, resp, ok := decodeNetworkChildProperties(body)
	if !ok {
		return resp, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, virtualNetworkName)
	vnet, exists := s.vnets[parentKey]
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual network %q could not be found.", virtualNetworkName))
	}

	subnet := Subnet{
		ID:         subnetID(subscriptionID, resourceGroup, virtualNetworkName, subnetName),
		Name:       subnetName,
		Type:       "Microsoft.Network/virtualNetworks/subnets",
		Properties: properties,
	}
	key := childKey(subscriptionID, resourceGroup, virtualNetworkName, subnetName)
	_, existed := s.subnets[key]
	s.subnets[key] = subnet
	vnet.Properties["subnets"] = s.subnetsForVirtualNetworkLocked(parentKey)
	s.vnets[parentKey] = vnet

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, subnet)
}

func (s *NetworkService) getSubnet(subscriptionID, resourceGroup, virtualNetworkName, subnetName string) (*service.Response, error) {
	s.mu.RLock()
	subnet, ok := s.subnets[childKey(subscriptionID, resourceGroup, virtualNetworkName, subnetName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Subnet %q could not be found.", subnetName))
	}
	return azurearm.JSONResponse(http.StatusOK, subnet)
}

func (s *NetworkService) listSubnets(subscriptionID, resourceGroup, virtualNetworkName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, virtualNetworkName)

	s.mu.RLock()
	_, parentExists := s.vnets[parentKey]
	values := s.subnetsForVirtualNetworkLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual network %q could not be found.", virtualNetworkName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteSubnet(subscriptionID, resourceGroup, virtualNetworkName, subnetName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, virtualNetworkName)
	vnet, parentExists := s.vnets[parentKey]
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Virtual network %q could not be found.", virtualNetworkName))
	}
	key := childKey(subscriptionID, resourceGroup, virtualNetworkName, subnetName)
	if _, ok := s.subnets[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Subnet %q could not be found.", subnetName))
	}
	delete(s.subnets, key)
	vnet.Properties["subnets"] = s.subnetsForVirtualNetworkLocked(parentKey)
	s.vnets[parentKey] = vnet
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateNetworkSecurityGroup(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
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
	id := networkSecurityGroupID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	input.Properties["securityRules"] = namedChildrenWithIDs(input.Properties["securityRules"], id+"/securityRules")
	if _, ok := input.Properties["defaultSecurityRules"]; !ok {
		input.Properties["defaultSecurityRules"] = []any{}
	}

	nsg := NetworkSecurityGroup{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/networkSecurityGroups",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.nsgs[key]
	s.nsgs[key] = nsg
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, nsg)
}

func (s *NetworkService) getNetworkSecurityGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	nsg, ok := s.nsgs[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network security group %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, nsg)
}

func (s *NetworkService) listNetworkSecurityGroups(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]NetworkSecurityGroup, 0)
	for key, nsg := range s.nsgs {
		if strings.HasPrefix(key, prefix) {
			values = append(values, nsg)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteNetworkSecurityGroup(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.nsgs[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network security group %q could not be found.", name))
	}
	delete(s.nsgs, key)
	childPrefix := key + "/"
	for ruleKey := range s.securityRules {
		if strings.HasPrefix(ruleKey, childPrefix) {
			delete(s.securityRules, ruleKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateSecurityRule(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName string, body []byte) (*service.Response, error) {
	properties, resp, ok := decodeNetworkChildProperties(body)
	if !ok {
		return resp, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, networkSecurityGroupName)
	nsg, exists := s.nsgs[parentKey]
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network security group %q could not be found.", networkSecurityGroupName))
	}

	rule := SecurityRule{
		ID:         securityRuleID(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName),
		Name:       ruleName,
		Type:       "Microsoft.Network/networkSecurityGroups/securityRules",
		Properties: properties,
	}
	key := childKey(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName)
	_, existed := s.securityRules[key]
	s.securityRules[key] = rule
	nsg.Properties["securityRules"] = s.securityRulesForNetworkSecurityGroupLocked(parentKey)
	s.nsgs[parentKey] = nsg

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, rule)
}

func (s *NetworkService) getSecurityRule(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName string) (*service.Response, error) {
	s.mu.RLock()
	rule, ok := s.securityRules[childKey(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Security rule %q could not be found.", ruleName))
	}
	return azurearm.JSONResponse(http.StatusOK, rule)
}

func (s *NetworkService) listSecurityRules(subscriptionID, resourceGroup, networkSecurityGroupName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, networkSecurityGroupName)

	s.mu.RLock()
	_, parentExists := s.nsgs[parentKey]
	values := s.securityRulesForNetworkSecurityGroupLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network security group %q could not be found.", networkSecurityGroupName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteSecurityRule(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, networkSecurityGroupName)
	nsg, parentExists := s.nsgs[parentKey]
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network security group %q could not be found.", networkSecurityGroupName))
	}
	key := childKey(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName)
	if _, ok := s.securityRules[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Security rule %q could not be found.", ruleName))
	}
	delete(s.securityRules, key)
	nsg.Properties["securityRules"] = s.securityRulesForNetworkSecurityGroupLocked(parentKey)
	s.nsgs[parentKey] = nsg
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdatePublicIPAddress(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
		Zones      []string       `json:"zones"`
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
	if _, ok := input.Properties["ipAddress"]; !ok {
		input.Properties["ipAddress"] = "203.0.113.10"
	}
	if dnsSettings, ok := input.Properties["dnsSettings"].(map[string]any); ok {
		if label := stringValue(dnsSettings["domainNameLabel"]); label != "" {
			enriched := cloneMap(dnsSettings)
			if _, ok := enriched["fqdn"]; !ok {
				enriched["fqdn"] = label + "." + input.Location + ".cloudapp.azure.com"
			}
			input.Properties["dnsSettings"] = enriched
		}
	}

	publicIP := PublicIPAddress{
		ID:         publicIPAddressID(subscriptionID, resourceGroup, name),
		Name:       name,
		Type:       "Microsoft.Network/publicIPAddresses",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Zones:      input.Zones,
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.publicIPs[key]
	s.publicIPs[key] = publicIP
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, publicIP)
}

func (s *NetworkService) getPublicIPAddress(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	publicIP, ok := s.publicIPs[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Public IP address %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, publicIP)
}

func (s *NetworkService) listPublicIPAddresses(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]PublicIPAddress, 0)
	for key, publicIP := range s.publicIPs {
		if strings.HasPrefix(key, prefix) {
			values = append(values, publicIP)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deletePublicIPAddress(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.publicIPs[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Public IP address %q could not be found.", name))
	}
	delete(s.publicIPs, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateNetworkInterface(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
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
	id := networkInterfaceID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	if _, ok := input.Properties["macAddress"]; !ok {
		input.Properties["macAddress"] = "00-0D-3A-00-00-01"
	}
	input.Properties["ipConfigurations"] = namedChildrenWithIDs(input.Properties["ipConfigurations"], id+"/ipConfigurations")

	networkInterface := NetworkInterface{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/networkInterfaces",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.networkInterfaces[key]
	s.networkInterfaces[key] = networkInterface
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, networkInterface)
}

func (s *NetworkService) getNetworkInterface(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	networkInterface, ok := s.networkInterfaces[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network interface %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, networkInterface)
}

func (s *NetworkService) listNetworkInterfaces(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]NetworkInterface, 0)
	for key, networkInterface := range s.networkInterfaces {
		if strings.HasPrefix(key, prefix) {
			values = append(values, networkInterface)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteNetworkInterface(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.networkInterfaces[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Network interface %q could not be found.", name))
	}
	delete(s.networkInterfaces, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateLoadBalancer(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
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
	id := loadBalancerID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	input.Properties["frontendIPConfigurations"] = namedChildrenWithIDs(input.Properties["frontendIPConfigurations"], id+"/frontendIPConfigurations")
	input.Properties["backendAddressPools"] = namedChildrenWithIDs(input.Properties["backendAddressPools"], id+"/backendAddressPools")
	input.Properties["probes"] = namedChildrenWithIDs(input.Properties["probes"], id+"/probes")
	input.Properties["loadBalancingRules"] = namedChildrenWithIDs(input.Properties["loadBalancingRules"], id+"/loadBalancingRules")
	input.Properties["inboundNatRules"] = namedChildrenWithIDs(input.Properties["inboundNatRules"], id+"/inboundNatRules")
	input.Properties["outboundRules"] = namedChildrenWithIDs(input.Properties["outboundRules"], id+"/outboundRules")

	loadBalancer := LoadBalancer{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/loadBalancers",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.loadBalancers[key]
	s.loadBalancers[key] = loadBalancer
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, loadBalancer)
}

func (s *NetworkService) getLoadBalancer(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	loadBalancer, ok := s.loadBalancers[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Load balancer %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, loadBalancer)
}

func (s *NetworkService) listLoadBalancers(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]LoadBalancer, 0)
	for key, loadBalancer := range s.loadBalancers {
		if strings.HasPrefix(key, prefix) {
			values = append(values, loadBalancer)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteLoadBalancer(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.loadBalancers[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Load balancer %q could not be found.", name))
	}
	delete(s.loadBalancers, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdateApplicationGateway(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		SKU        map[string]any `json:"sku"`
		Tags       map[string]any `json:"tags"`
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
	id := applicationGatewayID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	for _, collection := range []string{
		"authenticationCertificates",
		"backendAddressPools",
		"backendHttpSettingsCollection",
		"backendSettingsCollection",
		"frontendIPConfigurations",
		"frontendPorts",
		"gatewayIPConfigurations",
		"httpListeners",
		"listeners",
		"probes",
		"redirectConfigurations",
		"requestRoutingRules",
		"rewriteRuleSets",
		"routingRules",
		"sslCertificates",
		"sslProfiles",
		"trustedClientCertificates",
		"trustedRootCertificates",
		"urlPathMaps",
	} {
		input.Properties[collection] = namedChildrenWithIDs(input.Properties[collection], id+"/"+collection)
	}

	applicationGateway := ApplicationGateway{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/applicationGateways",
		Location:   input.Location,
		SKU:        input.SKU,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.applicationGateways[key]
	s.applicationGateways[key] = applicationGateway
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, applicationGateway)
}

func (s *NetworkService) getApplicationGateway(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	applicationGateway, ok := s.applicationGateways[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Application gateway %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, applicationGateway)
}

func (s *NetworkService) listApplicationGateways(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]ApplicationGateway, 0)
	for key, applicationGateway := range s.applicationGateways {
		if strings.HasPrefix(key, prefix) {
			values = append(values, applicationGateway)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deleteApplicationGateway(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.applicationGateways[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Application gateway %q could not be found.", name))
	}
	delete(s.applicationGateways, key)
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdatePrivateEndpoint(subscriptionID, resourceGroup, name string, body []byte) (*service.Response, error) {
	var input struct {
		Location   string         `json:"location"`
		Tags       map[string]any `json:"tags"`
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
	id := privateEndpointID(subscriptionID, resourceGroup, name)
	input.Properties["provisioningState"] = "Succeeded"
	input.Properties["privateLinkServiceConnections"] = namedChildrenWithIDs(input.Properties["privateLinkServiceConnections"], id+"/privateLinkServiceConnections")
	input.Properties["manualPrivateLinkServiceConnections"] = namedChildrenWithIDs(input.Properties["manualPrivateLinkServiceConnections"], id+"/manualPrivateLinkServiceConnections")
	input.Properties["ipConfigurations"] = namedChildrenWithIDs(input.Properties["ipConfigurations"], id+"/ipConfigurations")
	input.Properties["applicationSecurityGroups"] = namedChildrenWithIDs(input.Properties["applicationSecurityGroups"], id+"/applicationSecurityGroups")

	privateEndpoint := PrivateEndpoint{
		ID:         id,
		Name:       name,
		Type:       "Microsoft.Network/privateEndpoints",
		Location:   input.Location,
		Tags:       stringifyTags(input.Tags),
		Properties: input.Properties,
	}

	s.mu.Lock()
	key := resourceKey(subscriptionID, resourceGroup, name)
	_, existed := s.privateEndpoints[key]
	privateEndpoint.Properties["privateDnsZoneGroups"] = s.privateDNSZoneGroupsForPrivateEndpointLocked(key)
	s.privateEndpoints[key] = privateEndpoint
	s.mu.Unlock()

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, privateEndpoint)
}

func (s *NetworkService) getPrivateEndpoint(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.RLock()
	privateEndpoint, ok := s.privateEndpoints[resourceKey(subscriptionID, resourceGroup, name)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private endpoint %q could not be found.", name))
	}
	return azurearm.JSONResponse(http.StatusOK, privateEndpoint)
}

func (s *NetworkService) listPrivateEndpoints(subscriptionID, resourceGroup string) (*service.Response, error) {
	prefix := strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/"

	s.mu.RLock()
	values := make([]PrivateEndpoint, 0)
	for key, privateEndpoint := range s.privateEndpoints {
		if strings.HasPrefix(key, prefix) {
			values = append(values, privateEndpoint)
		}
	}
	s.mu.RUnlock()

	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deletePrivateEndpoint(subscriptionID, resourceGroup, name string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	key := resourceKey(subscriptionID, resourceGroup, name)
	if _, ok := s.privateEndpoints[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private endpoint %q could not be found.", name))
	}
	delete(s.privateEndpoints, key)
	childPrefix := key + "/"
	for zoneGroupKey := range s.privateDNSZoneGroups {
		if strings.HasPrefix(zoneGroupKey, childPrefix) {
			delete(s.privateDNSZoneGroups, zoneGroupKey)
		}
	}
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

func (s *NetworkService) createOrUpdatePrivateDNSZoneGroup(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName string, body []byte) (*service.Response, error) {
	properties, resp, ok := decodeNetworkChildProperties(body)
	if !ok {
		return resp, nil
	}
	id := privateDNSZoneGroupID(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName)
	properties["privateDnsZoneConfigs"] = namedChildrenWithIDs(properties["privateDnsZoneConfigs"], id+"/privateDnsZoneConfigs")

	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, privateEndpointName)
	privateEndpoint, exists := s.privateEndpoints[parentKey]
	if !exists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private endpoint %q could not be found.", privateEndpointName))
	}

	zoneGroup := PrivateDNSZoneGroup{
		ID:         id,
		Name:       zoneGroupName,
		Type:       "Microsoft.Network/privateEndpoints/privateDnsZoneGroups",
		Properties: properties,
	}
	key := childKey(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName)
	_, existed := s.privateDNSZoneGroups[key]
	s.privateDNSZoneGroups[key] = zoneGroup
	privateEndpoint.Properties["privateDnsZoneGroups"] = s.privateDNSZoneGroupsForPrivateEndpointLocked(parentKey)
	s.privateEndpoints[parentKey] = privateEndpoint

	status := http.StatusCreated
	if existed {
		status = http.StatusOK
	}
	return azurearm.JSONResponse(status, zoneGroup)
}

func (s *NetworkService) getPrivateDNSZoneGroup(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName string) (*service.Response, error) {
	s.mu.RLock()
	zoneGroup, ok := s.privateDNSZoneGroups[childKey(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName)]
	s.mu.RUnlock()
	if !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private DNS zone group %q could not be found.", zoneGroupName))
	}
	return azurearm.JSONResponse(http.StatusOK, zoneGroup)
}

func (s *NetworkService) listPrivateDNSZoneGroups(subscriptionID, resourceGroup, privateEndpointName string) (*service.Response, error) {
	parentKey := resourceKey(subscriptionID, resourceGroup, privateEndpointName)

	s.mu.RLock()
	_, parentExists := s.privateEndpoints[parentKey]
	values := s.privateDNSZoneGroupsForPrivateEndpointLocked(parentKey)
	s.mu.RUnlock()

	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private endpoint %q could not be found.", privateEndpointName))
	}
	return azurearm.JSONResponse(http.StatusOK, map[string]any{"value": values})
}

func (s *NetworkService) deletePrivateDNSZoneGroup(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName string) (*service.Response, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	parentKey := resourceKey(subscriptionID, resourceGroup, privateEndpointName)
	privateEndpoint, parentExists := s.privateEndpoints[parentKey]
	if !parentExists {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private endpoint %q could not be found.", privateEndpointName))
	}
	key := childKey(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName)
	if _, ok := s.privateDNSZoneGroups[key]; !ok {
		return azurearm.ErrorResponse(http.StatusNotFound, "ResourceNotFound", fmt.Sprintf("Private DNS zone group %q could not be found.", zoneGroupName))
	}
	delete(s.privateDNSZoneGroups, key)
	privateEndpoint.Properties["privateDnsZoneGroups"] = s.privateDNSZoneGroupsForPrivateEndpointLocked(parentKey)
	s.privateEndpoints[parentKey] = privateEndpoint
	return azurearm.JSONResponse(http.StatusAccepted, map[string]string{"status": "Succeeded"})
}

type networkRoute struct {
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
	Name           string
	ChildType      string
	ChildName      string
}

func parseRoute(escapedPath string) (networkRoute, bool) {
	parts := splitPath(escapedPath)
	if len(parts) < 7 ||
		!strings.EqualFold(parts[0], "subscriptions") ||
		!strings.EqualFold(parts[2], "resourceGroups") ||
		!strings.EqualFold(parts[4], "providers") ||
		!strings.EqualFold(parts[5], "Microsoft.Network") {
		return networkRoute{}, false
	}
	route := networkRoute{
		SubscriptionID: parts[1],
		ResourceGroup:  parts[3],
		ResourceType:   parts[6],
	}
	switch len(parts) {
	case 7:
		return route, true
	case 8:
		route.Name = parts[7]
		return route, true
	case 9:
		route.Name = parts[7]
		route.ChildType = parts[8]
		return route, true
	case 10:
		route.Name = parts[7]
		route.ChildType = parts[8]
		route.ChildName = parts[9]
		return route, true
	default:
		return networkRoute{}, false
	}
}

func decodeNetworkChildProperties(body []byte) (map[string]any, *service.Response, bool) {
	var input struct {
		Properties map[string]any `json:"properties"`
	}
	if len(body) > 0 {
		if err := gojson.Unmarshal(body, &input); err != nil {
			resp, _ := azurearm.ErrorResponse(http.StatusBadRequest, "InvalidRequestContent", "The request content was invalid.")
			return nil, resp, false
		}
	}
	if input.Properties == nil {
		input.Properties = make(map[string]any)
	}
	input.Properties["provisioningState"] = "Succeeded"
	return input.Properties, nil, true
}

func (s *NetworkService) subnetsForVirtualNetworkLocked(parentKey string) []Subnet {
	prefix := parentKey + "/"
	values := make([]Subnet, 0)
	for key, subnet := range s.subnets {
		if strings.HasPrefix(key, prefix) {
			values = append(values, subnet)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func (s *NetworkService) securityRulesForNetworkSecurityGroupLocked(parentKey string) []SecurityRule {
	prefix := parentKey + "/"
	values := make([]SecurityRule, 0)
	for key, rule := range s.securityRules {
		if strings.HasPrefix(key, prefix) {
			values = append(values, rule)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func (s *NetworkService) privateDNSZoneGroupsForPrivateEndpointLocked(parentKey string) []PrivateDNSZoneGroup {
	prefix := parentKey + "/"
	values := make([]PrivateDNSZoneGroup, 0)
	for key, zoneGroup := range s.privateDNSZoneGroups {
		if strings.HasPrefix(key, prefix) {
			values = append(values, zoneGroup)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].Name < values[j].Name })
	return values
}

func namedChildrenWithIDs(raw any, parentID string) []any {
	rawList, ok := raw.([]any)
	if !ok || len(rawList) == 0 {
		return []any{}
	}

	out := make([]any, 0, len(rawList))
	for _, item := range rawList {
		child, ok := item.(map[string]any)
		if !ok {
			continue
		}
		name := stringValue(child["name"])
		if name == "" {
			continue
		}
		enriched := cloneMap(child)
		enriched["id"] = parentID + "/" + name
		properties, _ := enriched["properties"].(map[string]any)
		if properties == nil {
			properties = make(map[string]any)
		} else {
			properties = cloneMap(properties)
		}
		properties["provisioningState"] = "Succeeded"
		enriched["properties"] = properties
		out = append(out, enriched)
	}
	return out
}

func virtualNetworkID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/virtualNetworks/" + name
}

func networkSecurityGroupID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkSecurityGroups/" + name
}

func publicIPAddressID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/publicIPAddresses/" + name
}

func networkInterfaceID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/networkInterfaces/" + name
}

func loadBalancerID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/loadBalancers/" + name
}

func applicationGatewayID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/applicationGateways/" + name
}

func privateEndpointID(subscriptionID, resourceGroup, name string) string {
	return "/subscriptions/" + subscriptionID + "/resourceGroups/" + resourceGroup + "/providers/Microsoft.Network/privateEndpoints/" + name
}

func privateDNSZoneGroupID(subscriptionID, resourceGroup, privateEndpointName, zoneGroupName string) string {
	return privateEndpointID(subscriptionID, resourceGroup, privateEndpointName) + "/privateDnsZoneGroups/" + zoneGroupName
}

func subnetID(subscriptionID, resourceGroup, virtualNetworkName, subnetName string) string {
	return virtualNetworkID(subscriptionID, resourceGroup, virtualNetworkName) + "/subnets/" + subnetName
}

func securityRuleID(subscriptionID, resourceGroup, networkSecurityGroupName, ruleName string) string {
	return networkSecurityGroupID(subscriptionID, resourceGroup, networkSecurityGroupName) + "/securityRules/" + ruleName
}

func resourceKey(subscriptionID, resourceGroup, name string) string {
	return strings.ToLower(subscriptionID) + "/" + strings.ToLower(resourceGroup) + "/" + strings.ToLower(name)
}

func childKey(subscriptionID, resourceGroup, parentName, childName string) string {
	return resourceKey(subscriptionID, resourceGroup, parentName) + "/" + strings.ToLower(childName)
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

func splitNestedName(name string) (string, string, bool) {
	parts := strings.Split(name, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

func cloneMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
