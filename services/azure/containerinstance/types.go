package containerinstance

// ContainerGroup is the ARM resource shape for Microsoft.ContainerInstance/containerGroups.
type ContainerGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// ContainerGroupProfile is the ARM resource shape for Microsoft.ContainerInstance/containerGroupProfiles.
type ContainerGroupProfile struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
	Zones      []any             `json:"zones,omitempty"`
}
