package compute

// VirtualMachine is the ARM resource shape for Microsoft.Compute/virtualMachines.
type VirtualMachine struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Identity   map[string]any    `json:"identity,omitempty"`
	Plan       map[string]any    `json:"plan,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Zones      []string          `json:"zones,omitempty"`
	Properties map[string]any    `json:"properties"`
}

type VirtualMachineInstanceView struct {
	Statuses []InstanceViewStatus `json:"statuses"`
}

type InstanceViewStatus struct {
	Code          string `json:"code"`
	Level         string `json:"level"`
	DisplayStatus string `json:"displayStatus"`
}

// Disk is the ARM resource shape for Microsoft.Compute/disks.
type Disk struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	SKU        map[string]any    `json:"sku,omitempty"`
	Tags       map[string]string `json:"tags,omitempty"`
	Zones      []string          `json:"zones,omitempty"`
	Properties map[string]any    `json:"properties"`
}
