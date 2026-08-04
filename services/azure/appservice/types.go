package appservice

// AppServicePlan is the ARM resource shape for Microsoft.Web/serverfarms.
type AppServicePlan struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Kind       string            `json:"kind,omitempty"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// Site is the ARM resource shape for Microsoft.Web/sites.
type Site struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Kind       string            `json:"kind,omitempty"`
	Location   string            `json:"location"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// Function is the ARM child resource shape returned by Web Apps - List Functions.
type Function struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Kind       string         `json:"kind,omitempty"`
	Location   string         `json:"location,omitempty"`
	Properties map[string]any `json:"properties"`
}

// SlotConfigNamesResource is the ARM resource shape for slot-sticky config names.
type SlotConfigNamesResource struct {
	ID         string                   `json:"id"`
	Name       string                   `json:"name"`
	Type       string                   `json:"type"`
	Kind       string                   `json:"kind,omitempty"`
	Properties SlotConfigNameProperties `json:"properties"`
}

// SlotConfigNameProperties stores config names that remain with deployment slots.
type SlotConfigNameProperties struct {
	AppSettingNames         []string `json:"appSettingNames"`
	ConnectionStringNames   []string `json:"connectionStringNames"`
	AzureStorageConfigNames []string `json:"azureStorageConfigNames"`
}

// LocalFunctionApp is the lightweight floci-compatible Functions admin app shape.
type LocalFunctionApp struct {
	Name           string            `json:"name"`
	Runtime        string            `json:"runtime"`
	LinuxFxVersion string            `json:"linuxFxVersion,omitempty"`
	Environment    map[string]string `json:"environment,omitempty"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"createdAt"`
}

// LocalFunction is the lightweight floci-compatible Functions admin function shape.
type LocalFunction struct {
	Name           string            `json:"name"`
	AppName        string            `json:"appName"`
	Runtime        string            `json:"runtime"`
	LinuxFxVersion string            `json:"linuxFxVersion,omitempty"`
	Handler        string            `json:"handler"`
	TimeoutSeconds int               `json:"timeoutSeconds"`
	Environment    map[string]string `json:"environment,omitempty"`
	InvokeURL      string            `json:"invokeUrl"`
	Status         string            `json:"status"`
	CreatedAt      string            `json:"createdAt"`
}
