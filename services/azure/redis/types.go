package redis

// Cache is the ARM resource shape for Microsoft.Cache/Redis.
type Cache struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties map[string]any    `json:"properties"`
}

type accessKeys struct {
	PrimaryKey   string `json:"primaryKey"`
	SecondaryKey string `json:"secondaryKey"`
}
