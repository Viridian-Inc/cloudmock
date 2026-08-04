package network

// VirtualNetwork is the ARM resource shape for Microsoft.Network/virtualNetworks.
type VirtualNetwork struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// NetworkSecurityGroup is the ARM resource shape for Microsoft.Network/networkSecurityGroups.
type NetworkSecurityGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// PublicIPAddress is the ARM resource shape for Microsoft.Network/publicIPAddresses.
type PublicIPAddress struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Zones      []string          `json:"zones,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// NetworkInterface is the ARM resource shape for Microsoft.Network/networkInterfaces.
type NetworkInterface struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// LoadBalancer is the ARM resource shape for Microsoft.Network/loadBalancers.
type LoadBalancer struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// ApplicationGateway is the ARM resource shape for Microsoft.Network/applicationGateways.
type ApplicationGateway struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// PrivateEndpoint is the ARM resource shape for Microsoft.Network/privateEndpoints.
type PrivateEndpoint struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// PrivateDNSZoneGroup is the ARM child resource shape for Microsoft.Network/privateEndpoints/privateDnsZoneGroups.
type PrivateDNSZoneGroup struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Subnet is the ARM child resource shape for Microsoft.Network/virtualNetworks/subnets.
type Subnet struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// SecurityRule is the ARM child resource shape for Microsoft.Network/networkSecurityGroups/securityRules.
type SecurityRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
