package cosmosdb

import "time"

// DatabaseAccount is the ARM resource shape for Microsoft.DocumentDB/databaseAccounts.
type DatabaseAccount struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// SQLDatabase is the ARM child resource shape for Microsoft.DocumentDB/databaseAccounts/sqlDatabases.
type SQLDatabase struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Options    map[string]any `json:"options,omitempty"`
}

// SQLContainer is the ARM child resource shape for Microsoft.DocumentDB/databaseAccounts/sqlDatabases/containers.
type SQLContainer struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
	Options    map[string]any `json:"options,omitempty"`
}

type cosmosDataDatabase struct {
	ID   string
	RID  string
	ETag string
	TS   int64
}

type cosmosDataCollection struct {
	ID             string
	RID            string
	ETag           string
	TS             int64
	PartitionKey   map[string]any
	IndexingPolicy map[string]any
}

type cosmosDataDocument struct {
	ID           string
	PartitionKey string
	RID          string
	ETag         string
	TS           int64
	Body         map[string]any
	UpdatedAt    time.Time
}
