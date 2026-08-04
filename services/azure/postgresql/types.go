package postgresql

// Server is the ARM resource shape for Microsoft.DBforPostgreSQL/flexibleServers.
type Server struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// Database is the ARM child resource shape for Microsoft.DBforPostgreSQL/flexibleServers/databases.
type Database struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// FirewallRule is the ARM child resource shape for Microsoft.DBforPostgreSQL/flexibleServers/firewallRules.
type FirewallRule struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
