package internal

import "time"

// Application represents an ArgoCD application.
type Application struct {
	Metadata AppMetadata       `json:"metadata"`
	Spec     ApplicationSpec   `json:"spec"`
	Status   ApplicationStatus `json:"status,omitempty"`
}

type AppMetadata struct {
	Name              string            `json:"name"`
	Namespace         string            `json:"namespace,omitempty"`
	UID               string            `json:"uid,omitempty"`
	ResourceVersion   string            `json:"resourceVersion,omitempty"`
	CreationTimestamp string            `json:"creationTimestamp,omitempty"`
	Labels            map[string]string `json:"labels,omitempty"`
	Annotations       map[string]string `json:"annotations,omitempty"`
}

type ApplicationSpec struct {
	Source      ApplicationSource      `json:"source"`
	Destination ApplicationDestination `json:"destination"`
	Project     string                 `json:"project,omitempty"`
	SyncPolicy  *SyncPolicy            `json:"syncPolicy,omitempty"`
}

type ApplicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path,omitempty"`
	TargetRevision string `json:"targetRevision,omitempty"`
	Chart          string `json:"chart,omitempty"`
}

type ApplicationDestination struct {
	Server    string `json:"server,omitempty"`
	Namespace string `json:"namespace,omitempty"`
	Name      string `json:"name,omitempty"`
}

type SyncPolicy struct {
	Automated *SyncPolicyAutomated `json:"automated,omitempty"`
}

type SyncPolicyAutomated struct {
	Prune    bool `json:"prune,omitempty"`
	SelfHeal bool `json:"selfHeal,omitempty"`
}

// ApplicationStatus holds the observed state of the application.
type ApplicationStatus struct {
	Sync           SyncStatus         `json:"sync"`
	Health         HealthStatus       `json:"health"`
	Summary        ApplicationSummary `json:"summary,omitempty"`
	Resources      []ResourceStatus   `json:"resources,omitempty"`
	OperationState *OperationState    `json:"operationState,omitempty"`
	ReconciledAt   string             `json:"reconciledAt,omitempty"`
}

type SyncStatus struct {
	Status   string `json:"status"` // Synced, OutOfSync, Unknown
	Revision string `json:"revision,omitempty"`
}

type HealthStatus struct {
	Status  string `json:"status"` // Healthy, Degraded, Progressing, Missing, Unknown
	Message string `json:"message,omitempty"`
}

type ApplicationSummary struct {
	Images []string `json:"images,omitempty"`
}

type ResourceStatus struct {
	Group     string        `json:"group,omitempty"`
	Version   string        `json:"version"`
	Kind      string        `json:"kind"`
	Namespace string        `json:"namespace,omitempty"`
	Name      string        `json:"name"`
	Status    string        `json:"status"` // Synced, OutOfSync
	Health    *HealthStatus `json:"health,omitempty"`
}

type OperationState struct {
	Operation  Operation `json:"operation"`
	Phase      string    `json:"phase"` // Running, Succeeded, Failed, Error
	Message    string    `json:"message,omitempty"`
	StartedAt  string    `json:"startedAt"`
	FinishedAt string    `json:"finishedAt,omitempty"`
}

type Operation struct {
	Sync *SyncOperation `json:"sync,omitempty"`
}

type SyncOperation struct {
	Revision string `json:"revision,omitempty"`
}

// ApplicationList is a list of applications.
type ApplicationList struct {
	Items    []Application `json:"items"`
	Metadata ListMetadata  `json:"metadata"`
}

type ListMetadata struct {
	ResourceVersion string `json:"resourceVersion,omitempty"`
}

// Repository represents a registered git repository.
type Repository struct {
	Repo            string          `json:"repo"`
	Type            string          `json:"type,omitempty"`
	Name            string          `json:"name,omitempty"`
	ConnectionState ConnectionState `json:"connectionState,omitempty"`
}

type ConnectionState struct {
	Status  string `json:"status"` // Successful, Failed
	Message string `json:"message,omitempty"`
}

// RepositoryList is a list of repositories.
type RepositoryList struct {
	Items    []Repository `json:"items"`
	Metadata ListMetadata `json:"metadata"`
}

// Cluster represents a registered cluster.
type Cluster struct {
	Server          string          `json:"server"`
	Name            string          `json:"name"`
	Config          ClusterConfig   `json:"config,omitempty"`
	ConnectionState ConnectionState `json:"connectionState,omitempty"`
}

type ClusterConfig struct {
	TLSClientConfig TLSClientConfig `json:"tlsClientConfig,omitempty"`
}

type TLSClientConfig struct {
	Insecure bool `json:"insecure,omitempty"`
}

// ClusterList is a list of clusters.
type ClusterList struct {
	Items    []Cluster    `json:"items"`
	Metadata ListMetadata `json:"metadata"`
}

// AppProject represents an ArgoCD project for multi-tenancy.
type AppProject struct {
	Metadata AppMetadata    `json:"metadata"`
	Spec     AppProjectSpec `json:"spec"`
}

type AppProjectSpec struct {
	Description              string                   `json:"description,omitempty"`
	SourceRepos              []string                 `json:"sourceRepos,omitempty"`
	SourceNamespaces         []string                 `json:"sourceNamespaces,omitempty"`
	Destinations             []ApplicationDestination `json:"destinations,omitempty"`
	ClusterResourceWhitelist []GroupKind              `json:"clusterResourceWhitelist,omitempty"`
}

type GroupKind struct {
	Group string `json:"group"`
	Kind  string `json:"kind"`
}

// AppProjectList is a list of projects.
type AppProjectList struct {
	Items    []AppProject `json:"items"`
	Metadata ListMetadata `json:"metadata"`
}

// Session holds login session info.
type Session struct {
	Token string `json:"token"`
}

// Settings holds ArgoCD server settings.
type Settings struct {
	URL               string   `json:"url,omitempty"`
	DexConfig         string   `json:"dexConfig,omitempty"`
	AppLabelKey       string   `json:"appLabelKey,omitempty"`
	KustomizeVersions []string `json:"kustomizeVersions,omitempty"`
}

// Now returns a timestamp string.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
