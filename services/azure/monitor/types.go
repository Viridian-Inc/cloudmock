package monitor

// ActionGroup is the ARM resource shape for Microsoft.Insights/actionGroups.
type ActionGroup struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// MetricAlert is the ARM resource shape for Microsoft.Insights/metricAlerts.
type MetricAlert struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}

// DiagnosticSetting is the ARM extension resource shape for Microsoft.Insights/diagnosticSettings.
type DiagnosticSetting struct {
	ID         string         `json:"id"`
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Properties map[string]any `json:"properties"`
}

// ApplicationInsightsComponent is the ARM resource shape for Microsoft.Insights/components.
type ApplicationInsightsComponent struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Kind       string            `json:"kind,omitempty"`
	Location   string            `json:"location"`
	Tags       map[string]string `json:"tags,omitempty"`
	Properties map[string]any    `json:"properties"`
}
