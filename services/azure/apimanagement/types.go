package apimanagement

// ServiceResource is the ARM shape for Microsoft.ApiManagement/service.
type ServiceResource struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Type           string            `json:"type"`
	Location       string            `json:"location"`
	Tags           map[string]string `json:"tags,omitempty"`
	SKU            map[string]any    `json:"sku"`
	Properties     map[string]any    `json:"properties"`
	SubscriptionID string            `json:"-"`
	ResourceGroup  string            `json:"-"`
}
