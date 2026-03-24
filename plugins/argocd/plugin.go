package argocd

import (
	"context"

	"github.com/neureaux/cloudmock/pkg/plugin"
	"github.com/neureaux/cloudmock/plugins/argocd/internal"
)

// K8sSync defines the interface the ArgoCD plugin needs from the Kubernetes
// plugin to create resources during a sync operation.
type K8sSync interface {
	// EnsureNamespace creates the namespace if it does not already exist.
	EnsureNamespace(name string)
	// EnsureDeployment creates a deployment if it does not already exist.
	EnsureDeployment(namespace, name string, labels map[string]string, matchLabels map[string]string)
}

// Plugin implements the CloudMock plugin interface for ArgoCD API emulation.
type Plugin struct {
	store   *internal.Store
	router  *internal.Router
	k8sSync K8sSync
}

// New creates a new ArgoCD plugin.
// k8sSync is optional — if provided, sync operations will apply resources
// to the k8s plugin's store for cross-plugin integration.
func New(k8sSync K8sSync) *Plugin {
	store := internal.NewStore("")
	p := &Plugin{
		store:   store,
		k8sSync: k8sSync,
	}
	p.router = internal.NewRouter(store, p.syncToK8s)
	return p
}

func (p *Plugin) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil
}

func (p *Plugin) Shutdown(_ context.Context) error {
	return nil
}

func (p *Plugin) HealthCheck(_ context.Context) (plugin.HealthStatus, string, error) {
	return plugin.HealthHealthy, "argocd api emulation is healthy", nil
}

func (p *Plugin) Describe(_ context.Context) (*plugin.Descriptor, error) {
	return &plugin.Descriptor{
		Name:     "argocd",
		Version:  "0.1.0",
		Protocol: "argocd-api",
		Actions: []string{
			"ListApplications", "GetApplication", "CreateApplication", "DeleteApplication", "SyncApplication",
			"ListRepositories", "CreateRepository", "DeleteRepository",
			"ListClusters",
			"ListProjects", "GetProject", "CreateProject",
			"CreateSession", "GetSettings",
		},
		APIPaths: []string{
			"/api/v1/applications",
			"/api/v1/repositories",
			"/api/v1/clusters",
			"/api/v1/projects",
			"/api/v1/session",
			"/api/v1/settings",
		},
		Metadata: map[string]string{
			"description": "ArgoCD API server emulation for CloudMock",
		},
	}, nil
}

func (p *Plugin) HandleRequest(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	statusCode, body := p.router.Handle(req.Method, req.Path, req.Body)
	return &plugin.Response{
		StatusCode: statusCode,
		Body:       body,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

// syncToK8s simulates applying an ArgoCD sync by creating resources in the k8s store.
func (p *Plugin) syncToK8s(app *internal.Application) {
	if p.k8sSync == nil {
		return
	}

	// Create a namespace for the app's destination if it doesn't exist.
	ns := app.Spec.Destination.Namespace
	if ns == "" {
		ns = "default"
	}
	p.k8sSync.EnsureNamespace(ns)

	// Create a deployment representing the synced application.
	depName := app.Metadata.Name
	p.k8sSync.EnsureDeployment(ns, depName,
		map[string]string{
			"app.kubernetes.io/instance":   app.Metadata.Name,
			"app.kubernetes.io/managed-by": "argocd",
		},
		map[string]string{"app": depName},
	)

	// Update app status with resource info.
	app.Status.Resources = []internal.ResourceStatus{
		{
			Version:   "apps/v1",
			Kind:      "Deployment",
			Namespace: ns,
			Name:      depName,
			Status:    "Synced",
			Health:    &internal.HealthStatus{Status: "Healthy"},
		},
	}
}

// Ensure Plugin satisfies the interface at compile time.
var _ plugin.Plugin = (*Plugin)(nil)
