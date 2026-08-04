package sql

// Server is the ARM resource shape for Microsoft.Sql/servers.
type Server struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// Database is the ARM child resource shape for Microsoft.Sql/servers/databases.
type Database struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Location   string         `json:"location,omitempty"`
	SKU        map[string]any `json:"sku,omitempty"`
	Properties map[string]any `json:"properties"`
}

// FirewallRule is the ARM child resource shape for Microsoft.Sql/servers/firewallRules.
type FirewallRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
