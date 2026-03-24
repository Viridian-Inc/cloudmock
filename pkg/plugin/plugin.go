// Package plugin defines the CloudMock plugin system.
//
// Plugins can be either in-process (Go) or out-of-process (any language via gRPC).
// Both kinds implement the same Plugin interface. The Manager discovers, loads,
// and routes requests to the appropriate plugin.
package plugin

import "context"

// HealthStatus represents the health state of a plugin.
type HealthStatus int

const (
	HealthUnknown   HealthStatus = 0
	HealthHealthy   HealthStatus = 1
	HealthDegraded  HealthStatus = 2
	HealthUnhealthy HealthStatus = 3
)

// String returns a human-readable representation of the health status.
func (h HealthStatus) String() string {
	switch h {
	case HealthHealthy:
		return "healthy"
	case HealthDegraded:
		return "degraded"
	case HealthUnhealthy:
		return "unhealthy"
	default:
		return "unknown"
	}
}

// Descriptor describes a plugin's identity and routing rules.
type Descriptor struct {
	Name     string            // e.g., "s3", "kubernetes", "argocd"
	Version  string            // semver
	Protocol string            // "aws-json", "aws-query", "k8s-api", "argocd-api"
	Actions  []string          // supported operations
	APIPaths []string          // URL path patterns for path-based routing
	Metadata map[string]string // arbitrary metadata
}

// Request carries an incoming HTTP request from the core to a plugin.
type Request struct {
	Action      string
	Body        []byte
	Headers     map[string]string
	QueryParams map[string]string
	Path        string
	Method      string
	Auth        *AuthContext
}

// Response carries a plugin's response back to the core.
type Response struct {
	StatusCode int
	Body       []byte
	Headers    map[string]string
}

// AuthContext carries the authenticated caller identity.
type AuthContext struct {
	UserID      string
	AccountID   string
	ARN         string
	AccessKeyID string
	IsRoot      bool
	Roles       []string
	Claims      map[string]string
}

// Plugin is the interface that all CloudMock plugins implement.
type Plugin interface {
	// Init is called once when the plugin is loaded.
	Init(ctx context.Context, config []byte, dataDir string, logLevel string) error

	// Shutdown is called when the core is stopping.
	Shutdown(ctx context.Context) error

	// HealthCheck returns the plugin's current health status.
	HealthCheck(ctx context.Context) (HealthStatus, string, error)

	// Describe returns the plugin's metadata and routing rules.
	Describe(ctx context.Context) (*Descriptor, error)

	// HandleRequest processes a single request.
	HandleRequest(ctx context.Context, req *Request) (*Response, error)
}

// StreamingPlugin is an optional interface for plugins that support streaming
// responses (e.g., Kubernetes watch).
type StreamingPlugin interface {
	Plugin
	// StreamRequest processes a request that produces multiple responses.
	StreamRequest(ctx context.Context, req *Request, send func(*Response) error) error
}
