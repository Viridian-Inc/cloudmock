package cdn

// Profile is the ARM resource shape for Microsoft.Cdn/profiles.
type Profile struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	SKU        ProfileSKU        `json:"sku"`
	Kind       string            `json:"kind"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Properties ProfileProperties `json:"properties"`
}

type ProfileSKU struct {
	Name string `json:"name"`
}

type ProfileProperties struct {
	OriginResponseTimeoutSeconds int            `json:"originResponseTimeoutSeconds"`
	LogScrubbing                 map[string]any `json:"logScrubbing,omitempty"`
	ExtendedProperties           map[string]any `json:"extendedProperties,omitempty"`
	FrontDoorID                  string         `json:"frontDoorId"`
	ProvisioningState            string         `json:"provisioningState"`
	ResourceState                string         `json:"resourceState"`
}

// Endpoint is the ARM resource shape for Microsoft.Cdn/profiles/endpoints.
type Endpoint struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// OriginGroup is the ARM resource shape for Microsoft.Cdn/profiles/endpoints/originGroups.
type OriginGroup struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// Origin is the ARM resource shape for Microsoft.Cdn/profiles/endpoints/origins.
type Origin struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// CustomDomain is the ARM resource shape for Microsoft.Cdn/profiles/endpoints/customDomains.
type CustomDomain struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}
