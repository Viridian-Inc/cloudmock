// Package marketplace provides a plugin marketplace registry for CloudMock.
// It allows searching, listing, and (placeholder) installing/uninstalling plugins.
package marketplace

import (
	"strings"
	"sync"
)

// PluginListing describes a plugin available in the marketplace.
type PluginListing struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Author      string  `json:"author"`
	Version     string  `json:"version"`
	Category    string  `json:"category"` // "aws", "gcp", "azure", "integration", "tool"
	Downloads   int     `json:"downloads"`
	Rating      float64 `json:"rating"`
	RepoURL     string  `json:"repo_url"`
	InstallCmd  string  `json:"install_cmd"`
	Installed   bool    `json:"installed"`
}

// Registry holds the marketplace plugin listings.
type Registry struct {
	mu       sync.RWMutex
	listings []PluginListing
}

// NewRegistry creates a registry seeded with built-in and community listings.
func NewRegistry() *Registry {
	r := &Registry{}
	r.listings = seedListings()
	return r
}

// Search filters listings by query string and category.
// Both query and category are optional (empty = no filter).
func (r *Registry) Search(query string, category string) []PluginListing {
	r.mu.RLock()
	defer r.mu.RUnlock()

	query = strings.ToLower(strings.TrimSpace(query))
	category = strings.ToLower(strings.TrimSpace(category))

	var out []PluginListing
	for _, l := range r.listings {
		if category != "" && !strings.EqualFold(l.Category, category) {
			continue
		}
		if query != "" {
			name := strings.ToLower(l.Name)
			desc := strings.ToLower(l.Description)
			if !strings.Contains(name, query) && !strings.Contains(desc, query) {
				continue
			}
		}
		out = append(out, l)
	}
	return out
}

// Get returns a listing by ID.
func (r *Registry) Get(id string) (*PluginListing, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for i := range r.listings {
		if r.listings[i].ID == id {
			cp := r.listings[i]
			return &cp, true
		}
	}
	return nil, false
}

// Install marks a plugin as installed (placeholder).
func (r *Registry) Install(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.listings {
		if r.listings[i].ID == id {
			r.listings[i].Installed = true
			r.listings[i].Downloads++
			return nil
		}
	}
	return ErrNotFound
}

// Uninstall marks a plugin as not installed (placeholder).
func (r *Registry) Uninstall(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for i := range r.listings {
		if r.listings[i].ID == id {
			r.listings[i].Installed = false
			return nil
		}
	}
	return ErrNotFound
}

// List returns all listings.
func (r *Registry) List() []PluginListing {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]PluginListing, len(r.listings))
	copy(out, r.listings)
	return out
}

// ErrNotFound is returned when a plugin listing is not found.
var ErrNotFound = &marketplaceError{"plugin not found"}

type marketplaceError struct {
	msg string
}

func (e *marketplaceError) Error() string { return e.msg }

func seedListings() []PluginListing {
	return []PluginListing{
		// Built-in plugins (installed by default)
		{
			ID:          "kubernetes",
			Name:        "Kubernetes",
			Description: "Kubernetes API emulation — pods, deployments, services, namespaces",
			Author:      "CloudMock",
			Version:     "1.0.0",
			Category:    "integration",
			Downloads:   1250,
			Rating:      4.8,
			RepoURL:     "https://github.com/neureaux/cloudmock/tree/main/plugins/kubernetes",
			InstallCmd:  "built-in",
			Installed:   true,
		},
		{
			ID:          "argocd",
			Name:        "ArgoCD",
			Description: "ArgoCD GitOps continuous delivery — applications, sync, rollback",
			Author:      "CloudMock",
			Version:     "1.0.0",
			Category:    "integration",
			Downloads:   890,
			Rating:      4.7,
			RepoURL:     "https://github.com/neureaux/cloudmock/tree/main/plugins/argocd",
			InstallCmd:  "built-in",
			Installed:   true,
		},
		{
			ID:          "flyio",
			Name:        "Fly.io Machines",
			Description: "Fly.io Machines API emulation — apps, machines, volumes, secrets",
			Author:      "CloudMock",
			Version:     "1.0.0",
			Category:    "integration",
			Downloads:   680,
			Rating:      4.6,
			RepoURL:     "https://github.com/neureaux/cloudmock/tree/main/plugins/flyio",
			InstallCmd:  "built-in",
			Installed:   true,
		},
		{
			ID:          "example-external",
			Name:        "Example External Plugin",
			Description: "Reference implementation for building external CloudMock plugins via gRPC",
			Author:      "CloudMock",
			Version:     "1.0.0",
			Category:    "tool",
			Downloads:   320,
			Rating:      4.2,
			RepoURL:     "https://github.com/neureaux/cloudmock/tree/main/plugins/example",
			InstallCmd:  "built-in",
			Installed:   true,
		},
		// Community / placeholder listings
		{
			ID:          "terraform-state",
			Name:        "Terraform State Backend",
			Description: "Mock Terraform state backend with locking — use CloudMock as your tfstate store",
			Author:      "community",
			Version:     "0.3.0",
			Category:    "tool",
			Downloads:   540,
			Rating:      4.3,
			RepoURL:     "https://github.com/cloudmock-community/terraform-state-plugin",
			InstallCmd:  "cloudmock plugin install terraform-state",
		},
		{
			ID:          "gcp-emulator",
			Name:        "GCP Service Emulator",
			Description: "Google Cloud Platform service mocks — Pub/Sub, GCS, BigQuery, Firestore",
			Author:      "community",
			Version:     "0.2.0",
			Category:    "gcp",
			Downloads:   410,
			Rating:      4.1,
			RepoURL:     "https://github.com/cloudmock-community/gcp-emulator",
			InstallCmd:  "cloudmock plugin install gcp-emulator",
		},
		{
			ID:          "azure-emulator",
			Name:        "Azure Service Emulator",
			Description: "Microsoft Azure service mocks — Blob Storage, Cosmos DB, Service Bus",
			Author:      "community",
			Version:     "0.1.0",
			Category:    "azure",
			Downloads:   180,
			Rating:      3.8,
			RepoURL:     "https://github.com/cloudmock-community/azure-emulator",
			InstallCmd:  "cloudmock plugin install azure-emulator",
		},
		{
			ID:          "datadog-forwarder",
			Name:        "Datadog Forwarder",
			Description: "Forward CloudMock telemetry to Datadog — metrics, traces, and logs",
			Author:      "community",
			Version:     "0.4.0",
			Category:    "integration",
			Downloads:   670,
			Rating:      4.5,
			RepoURL:     "https://github.com/cloudmock-community/datadog-forwarder",
			InstallCmd:  "cloudmock plugin install datadog-forwarder",
		},
		{
			ID:          "grafana-datasource",
			Name:        "Grafana Data Source",
			Description: "Grafana data source plugin for querying CloudMock metrics and traces",
			Author:      "community",
			Version:     "0.3.0",
			Category:    "integration",
			Downloads:   520,
			Rating:      4.4,
			RepoURL:     "https://github.com/cloudmock-community/grafana-datasource",
			InstallCmd:  "cloudmock plugin install grafana-datasource",
		},
		{
			ID:          "chaos-monkey",
			Name:        "Chaos Monkey",
			Description: "Automated chaos engineering — random fault injection, latency spikes, service kills",
			Author:      "community",
			Version:     "0.2.0",
			Category:    "tool",
			Downloads:   350,
			Rating:      4.0,
			RepoURL:     "https://github.com/cloudmock-community/chaos-monkey",
			InstallCmd:  "cloudmock plugin install chaos-monkey",
		},
		{
			ID:          "slack-notifier",
			Name:        "Slack Notifier",
			Description: "Send CloudMock alerts and findings to Slack channels",
			Author:      "community",
			Version:     "0.5.0",
			Category:    "integration",
			Downloads:   780,
			Rating:      4.6,
			RepoURL:     "https://github.com/cloudmock-community/slack-notifier",
			InstallCmd:  "cloudmock plugin install slack-notifier",
		},
	}
}
