package kubernetes

import (
	"context"

	"github.com/neureaux/cloudmock/pkg/plugin"
	"github.com/neureaux/cloudmock/plugins/kubernetes/internal"
)

// Plugin implements the CloudMock plugin interface for Kubernetes API emulation.
type Plugin struct {
	store  *internal.Store
	router *internal.Router
}

// New creates a new Kubernetes plugin.
func New() *Plugin {
	store := internal.NewStore()
	return &Plugin{
		store:  store,
		router: internal.NewRouter(store),
	}
}

func (p *Plugin) Init(_ context.Context, _ []byte, _ string, _ string) error {
	return nil
}

func (p *Plugin) Shutdown(_ context.Context) error {
	return nil
}

func (p *Plugin) HealthCheck(_ context.Context) (plugin.HealthStatus, string, error) {
	return plugin.HealthHealthy, "kubernetes api server emulation is healthy", nil
}

func (p *Plugin) Describe(_ context.Context) (*plugin.Descriptor, error) {
	return &plugin.Descriptor{
		Name:     "kubernetes",
		Version:  "0.1.0",
		Protocol: "k8s-api",
		Actions: []string{
			"GetNamespaces", "CreateNamespace", "DeleteNamespace",
			"GetPods", "CreatePod", "DeletePod",
			"GetServices", "CreateService", "DeleteService",
			"GetConfigMaps", "CreateConfigMap", "DeleteConfigMap",
			"GetSecrets", "CreateSecret", "DeleteSecret",
			"GetDeployments", "CreateDeployment", "DeleteDeployment",
			"GetNodes",
		},
		APIPaths: []string{
			"/api/",
			"/api/v1/",
			"/apis/",
			"/version",
		},
		Metadata: map[string]string{
			"description": "Kubernetes API server emulation for CloudMock",
		},
	}, nil
}

func (p *Plugin) HandleRequest(_ context.Context, req *plugin.Request) (*plugin.Response, error) {
	statusCode, body := p.router.Handle(req.Method, req.Path, req.Body, req.QueryParams)
	return &plugin.Response{
		StatusCode: statusCode,
		Body:       body,
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
	}, nil
}

// Store returns the internal store for cross-plugin access (e.g., ArgoCD sync).
func (p *Plugin) Store() *internal.Store {
	return p.store
}

// EnsureNamespace creates the namespace if it does not already exist.
// This is used by the ArgoCD plugin during sync operations.
func (p *Plugin) EnsureNamespace(name string) {
	if _, exists := p.store.GetNamespace(name); !exists {
		p.store.CreateNamespace(&internal.Namespace{
			TypeMeta:   internal.TypeMeta{Kind: "Namespace", APIVersion: "v1"},
			ObjectMeta: internal.ObjectMeta{Name: name},
		})
	}
}

// EnsureDeployment creates a deployment if it does not already exist.
// This is used by the ArgoCD plugin during sync operations.
func (p *Plugin) EnsureDeployment(namespace, name string, labels map[string]string, matchLabels map[string]string) {
	if _, exists := p.store.GetDeployment(namespace, name); !exists {
		p.store.CreateDeployment(&internal.Deployment{
			TypeMeta: internal.TypeMeta{Kind: "Deployment", APIVersion: "apps/v1"},
			ObjectMeta: internal.ObjectMeta{
				Name:      name,
				Namespace: namespace,
				Labels:    labels,
			},
			Spec: internal.DeploymentSpec{
				Replicas: 1,
				Selector: &internal.LabelSelector{
					MatchLabels: matchLabels,
				},
			},
		})
	}
}

// Ensure Plugin satisfies the interface at compile time.
var _ plugin.Plugin = (*Plugin)(nil)
