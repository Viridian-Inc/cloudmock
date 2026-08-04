package appconfiguration

// ConfigurationStore is the ARM resource shape for Microsoft.AppConfiguration/configurationStores.
type ConfigurationStore struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties map[string]any    `json:"properties"`
}

type keyValue struct {
	Key          string
	Label        string
	Value        any
	ContentType  any
	Tags         map[string]any
	ETag         string
	LastModified string
	Locked       bool
}

type snapshot struct {
	Name             string
	Status           string
	Filters          []snapshotFilter
	CompositionType  string
	Created          string
	Expires          any
	RetentionSeconds float64
	ItemsCount       int
	Size             int
	Tags             map[string]string
	ETag             string
	Items            []keyValue
}

type snapshotFilter struct {
	Key   string   `json:"key"`
	Label string   `json:"label"`
	Tags  []string `json:"tags,omitempty"`
}
