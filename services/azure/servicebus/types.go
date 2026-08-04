package servicebus

// Namespace is the ARM resource shape for Microsoft.ServiceBus/namespaces.
type Namespace struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// AuthorizationRule is the ARM child resource shape for Microsoft.ServiceBus/namespaces/AuthorizationRules.
type AuthorizationRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// AccessKeys is the response shape for Service Bus namespace authorization rule keys.
type AccessKeys struct {
	KeyName                   string `json:"keyName"`
	PrimaryConnectionString   string `json:"primaryConnectionString"`
	PrimaryKey                string `json:"primaryKey"`
	SecondaryConnectionString string `json:"secondaryConnectionString"`
	SecondaryKey              string `json:"secondaryKey"`
}

// Queue is the ARM child resource shape for Microsoft.ServiceBus/namespaces/queues.
type Queue struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Topic is the ARM child resource shape for Microsoft.ServiceBus/namespaces/topics.
type Topic struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Subscription is the ARM child resource shape for Microsoft.ServiceBus/namespaces/topics/subscriptions.
type Subscription struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Rule is the ARM child resource shape for Microsoft.ServiceBus/namespaces/topics/subscriptions/rules.
type Rule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
