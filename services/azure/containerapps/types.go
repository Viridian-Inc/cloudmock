package containerapps

// ManagedEnvironment is the ARM resource shape for Microsoft.App/managedEnvironments.
type ManagedEnvironment struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// ContainerApp is the ARM resource shape for Microsoft.App/containerApps.
type ContainerApp struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Kind       string            `json:"kind,omitempty"`
	Properties map[string]any    `json:"properties"`
}
