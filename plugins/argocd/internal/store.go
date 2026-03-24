package internal

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
)

// Store holds all ArgoCD resources in memory.
type Store struct {
	mu              sync.RWMutex
	apps            map[string]*Application // name -> app
	repos           map[string]*Repository  // url -> repo
	clusters        map[string]*Cluster     // server -> cluster
	projects        map[string]*AppProject  // name -> project
	resourceVersion atomic.Int64
}

// NewStore creates a store with default project and in-cluster.
func NewStore(gatewayAddr string) *Store {
	s := &Store{
		apps:     make(map[string]*Application),
		repos:    make(map[string]*Repository),
		clusters: make(map[string]*Cluster),
		projects: make(map[string]*AppProject),
	}
	s.resourceVersion.Store(1)

	// Default project.
	s.projects["default"] = &AppProject{
		Metadata: AppMetadata{Name: "default", UID: generateUID(), ResourceVersion: "1", CreationTimestamp: Now()},
		Spec: AppProjectSpec{
			Description: "Default project",
			SourceRepos: []string{"*"},
			Destinations: []ApplicationDestination{{Server: "*", Namespace: "*"}},
			ClusterResourceWhitelist: []GroupKind{{Group: "*", Kind: "*"}},
		},
	}

	// In-cluster pointing to cloudmock's own k8s plugin.
	server := "https://kubernetes.default.svc"
	if gatewayAddr != "" {
		server = gatewayAddr
	}
	s.clusters[server] = &Cluster{
		Server:          server,
		Name:            "in-cluster",
		ConnectionState: ConnectionState{Status: "Successful", Message: "ok"},
	}

	return s
}

func (s *Store) nextVersion() string {
	return strconv.FormatInt(s.resourceVersion.Add(1), 10)
}

var argoUIDCounter atomic.Int64

func generateUID() string {
	return fmt.Sprintf("argo-%d", argoUIDCounter.Add(1))
}

// --- Applications ---

func (s *Store) GetApp(name string) (*Application, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	app, ok := s.apps[name]
	return app, ok
}

func (s *Store) ListApps() []Application {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Application, 0, len(s.apps))
	for _, app := range s.apps {
		out = append(out, *app)
	}
	return out
}

func (s *Store) CreateApp(app *Application) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.apps[app.Metadata.Name]; exists {
		return fmt.Errorf("application %q already exists", app.Metadata.Name)
	}
	app.Metadata.UID = generateUID()
	app.Metadata.ResourceVersion = s.nextVersion()
	app.Metadata.CreationTimestamp = Now()
	if app.Spec.Project == "" {
		app.Spec.Project = "default"
	}
	app.Status = ApplicationStatus{
		Sync:   SyncStatus{Status: "OutOfSync"},
		Health: HealthStatus{Status: "Missing"},
	}
	s.apps[app.Metadata.Name] = app
	return nil
}

func (s *Store) UpdateApp(app *Application) {
	s.mu.Lock()
	defer s.mu.Unlock()
	app.Metadata.ResourceVersion = s.nextVersion()
	s.apps[app.Metadata.Name] = app
}

func (s *Store) DeleteApp(name string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.apps[name]; !exists {
		return fmt.Errorf("application %q not found", name)
	}
	delete(s.apps, name)
	return nil
}

// --- Repositories ---

func (s *Store) GetRepo(url string) (*Repository, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	repo, ok := s.repos[url]
	return repo, ok
}

func (s *Store) ListRepos() []Repository {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Repository, 0, len(s.repos))
	for _, r := range s.repos {
		out = append(out, *r)
	}
	return out
}

func (s *Store) CreateRepo(repo *Repository) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.repos[repo.Repo]; exists {
		return fmt.Errorf("repository %q already exists", repo.Repo)
	}
	repo.ConnectionState = ConnectionState{Status: "Successful", Message: "ok"}
	if repo.Type == "" {
		repo.Type = "git"
	}
	s.repos[repo.Repo] = repo
	return nil
}

func (s *Store) DeleteRepo(url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.repos[url]; !exists {
		return fmt.Errorf("repository %q not found", url)
	}
	delete(s.repos, url)
	return nil
}

// --- Clusters ---

func (s *Store) ListClusters() []Cluster {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Cluster, 0, len(s.clusters))
	for _, c := range s.clusters {
		out = append(out, *c)
	}
	return out
}

// --- Projects ---

func (s *Store) GetProject(name string) (*AppProject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	proj, ok := s.projects[name]
	return proj, ok
}

func (s *Store) ListProjects() []AppProject {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]AppProject, 0, len(s.projects))
	for _, p := range s.projects {
		out = append(out, *p)
	}
	return out
}

func (s *Store) CreateProject(proj *AppProject) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.projects[proj.Metadata.Name]; exists {
		return fmt.Errorf("project %q already exists", proj.Metadata.Name)
	}
	proj.Metadata.UID = generateUID()
	proj.Metadata.ResourceVersion = s.nextVersion()
	proj.Metadata.CreationTimestamp = Now()
	s.projects[proj.Metadata.Name] = proj
	return nil
}
