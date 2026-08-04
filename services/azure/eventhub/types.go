package eventhub

// Namespace is the ARM resource shape for Microsoft.EventHub/namespaces.
type Namespace struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// AuthorizationRule is the ARM child resource shape for Microsoft.EventHub/namespaces/authorizationRules.
type AuthorizationRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// AccessKeys is the response shape for Event Hubs namespace authorization rule keys.
type AccessKeys struct {
	KeyName                   string `json:"keyName"`
	PrimaryConnectionString   string `json:"primaryConnectionString"`
	PrimaryKey                string `json:"primaryKey"`
	SecondaryConnectionString string `json:"secondaryConnectionString"`
	SecondaryKey              string `json:"secondaryKey"`
}

// EventHub is the ARM child resource shape for Microsoft.EventHub/namespaces/eventhubs.
type EventHub struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// ConsumerGroup is the ARM child resource shape for Microsoft.EventHub/namespaces/eventhubs/consumergroups.
type ConsumerGroup struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
