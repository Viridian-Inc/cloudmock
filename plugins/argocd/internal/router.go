package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Router handles ArgoCD API requests.
type Router struct {
	store  *Store
	syncFn func(app *Application) // callback to sync to k8s plugin
}

// NewRouter creates an ArgoCD API router.
func NewRouter(store *Store, syncFn func(app *Application)) *Router {
	return &Router{store: store, syncFn: syncFn}
}

// Handle routes a request to the appropriate handler.
func (rt *Router) Handle(method, path string, body []byte) (int, []byte) {
	path = strings.TrimRight(path, "/")

	// /api/v1/session
	if path == "/api/v1/session" {
		return rt.handleSession(method, body)
	}
	// /api/v1/settings
	if path == "/api/v1/settings" {
		return jsonResp(200, Settings{AppLabelKey: "app.kubernetes.io/instance"})
	}

	// /api/v1/applications
	if path == "/api/v1/applications" {
		return rt.handleApplications(method, body)
	}
	// /api/v1/applications/{name}
	if strings.HasPrefix(path, "/api/v1/applications/") {
		rest := strings.TrimPrefix(path, "/api/v1/applications/")
		parts := strings.SplitN(rest, "/", 2)
		name := parts[0]
		if len(parts) == 2 && parts[1] == "sync" {
			return rt.handleSync(name, body)
		}
		if len(parts) == 2 && parts[1] == "resource-tree" {
			return rt.handleResourceTree(name)
		}
		return rt.handleApplicationByName(method, name, body)
	}

	// /api/v1/repositories
	if path == "/api/v1/repositories" {
		return rt.handleRepositories(method, body)
	}
	if strings.HasPrefix(path, "/api/v1/repositories/") {
		url := strings.TrimPrefix(path, "/api/v1/repositories/")
		return rt.handleRepositoryByURL(method, url)
	}

	// /api/v1/clusters
	if path == "/api/v1/clusters" {
		return rt.handleClusters(method)
	}

	// /api/v1/projects
	if path == "/api/v1/projects" {
		return rt.handleProjects(method, body)
	}
	if strings.HasPrefix(path, "/api/v1/projects/") {
		name := strings.TrimPrefix(path, "/api/v1/projects/")
		return rt.handleProjectByName(method, name)
	}

	return errorResp(404, "not found: "+path)
}

// --- Session ---

func (rt *Router) handleSession(method string, body []byte) (int, []byte) {
	if method == "POST" {
		return jsonResp(200, Session{Token: "cloudmock-argocd-token"})
	}
	return errorResp(405, "method not allowed")
}

// --- Applications ---

func (rt *Router) handleApplications(method string, body []byte) (int, []byte) {
	switch method {
	case "GET":
		apps := rt.store.ListApps()
		return jsonResp(200, ApplicationList{Items: apps})
	case "POST":
		var app Application
		if err := json.Unmarshal(body, &app); err != nil {
			return errorResp(400, err.Error())
		}
		if err := rt.store.CreateApp(&app); err != nil {
			return errorResp(409, err.Error())
		}
		return jsonResp(200, app)
	}
	return errorResp(405, "method not allowed")
}

func (rt *Router) handleApplicationByName(method, name string, body []byte) (int, []byte) {
	switch method {
	case "GET":
		app, ok := rt.store.GetApp(name)
		if !ok {
			return errorResp(404, fmt.Sprintf("application %q not found", name))
		}
		return jsonResp(200, app)
	case "DELETE":
		if err := rt.store.DeleteApp(name); err != nil {
			return errorResp(404, err.Error())
		}
		return jsonResp(200, map[string]string{"status": "ok"})
	}
	return errorResp(405, "method not allowed")
}

func (rt *Router) handleSync(name string, body []byte) (int, []byte) {
	app, ok := rt.store.GetApp(name)
	if !ok {
		return errorResp(404, fmt.Sprintf("application %q not found", name))
	}

	// Simulate sync.
	app.Status.Sync = SyncStatus{Status: "Synced", Revision: "HEAD"}
	app.Status.Health = HealthStatus{Status: "Healthy", Message: "all resources healthy"}
	app.Status.ReconciledAt = Now()
	app.Status.OperationState = &OperationState{
		Operation:  Operation{Sync: &SyncOperation{Revision: "HEAD"}},
		Phase:      "Succeeded",
		Message:    "successfully synced (all tasks run)",
		StartedAt:  Now(),
		FinishedAt: Now(),
	}

	// Call the sync callback to apply to k8s if available.
	if rt.syncFn != nil {
		rt.syncFn(app)
	}

	rt.store.UpdateApp(app)
	return jsonResp(200, app)
}

func (rt *Router) handleResourceTree(name string) (int, []byte) {
	app, ok := rt.store.GetApp(name)
	if !ok {
		return errorResp(404, fmt.Sprintf("application %q not found", name))
	}
	// Return a simple resource tree.
	tree := map[string]interface{}{
		"nodes": app.Status.Resources,
	}
	return jsonResp(200, tree)
}

// --- Repositories ---

func (rt *Router) handleRepositories(method string, body []byte) (int, []byte) {
	switch method {
	case "GET":
		repos := rt.store.ListRepos()
		return jsonResp(200, RepositoryList{Items: repos})
	case "POST":
		var repo Repository
		if err := json.Unmarshal(body, &repo); err != nil {
			return errorResp(400, err.Error())
		}
		if err := rt.store.CreateRepo(&repo); err != nil {
			return errorResp(409, err.Error())
		}
		return jsonResp(200, repo)
	}
	return errorResp(405, "method not allowed")
}

func (rt *Router) handleRepositoryByURL(method, url string) (int, []byte) {
	switch method {
	case "GET":
		repo, ok := rt.store.GetRepo(url)
		if !ok {
			return errorResp(404, fmt.Sprintf("repository %q not found", url))
		}
		return jsonResp(200, repo)
	case "DELETE":
		if err := rt.store.DeleteRepo(url); err != nil {
			return errorResp(404, err.Error())
		}
		return jsonResp(200, map[string]string{"status": "ok"})
	}
	return errorResp(405, "method not allowed")
}

// --- Clusters ---

func (rt *Router) handleClusters(method string) (int, []byte) {
	if method == "GET" {
		clusters := rt.store.ListClusters()
		return jsonResp(200, ClusterList{Items: clusters})
	}
	return errorResp(405, "method not allowed")
}

// --- Projects ---

func (rt *Router) handleProjects(method string, body []byte) (int, []byte) {
	switch method {
	case "GET":
		projects := rt.store.ListProjects()
		return jsonResp(200, AppProjectList{Items: projects})
	case "POST":
		var proj AppProject
		if err := json.Unmarshal(body, &proj); err != nil {
			return errorResp(400, err.Error())
		}
		if err := rt.store.CreateProject(&proj); err != nil {
			return errorResp(409, err.Error())
		}
		return jsonResp(200, proj)
	}
	return errorResp(405, "method not allowed")
}

func (rt *Router) handleProjectByName(method, name string) (int, []byte) {
	switch method {
	case "GET":
		proj, ok := rt.store.GetProject(name)
		if !ok {
			return errorResp(404, fmt.Sprintf("project %q not found", name))
		}
		return jsonResp(200, proj)
	}
	return errorResp(405, "method not allowed")
}

// --- Helpers ---

func jsonResp(status int, v interface{}) (int, []byte) {
	data, err := json.Marshal(v)
	if err != nil {
		return 500, []byte(`{"error":"internal marshal error"}`)
	}
	return status, data
}

func errorResp(code int, message string) (int, []byte) {
	return jsonResp(code, map[string]interface{}{
		"error":   message,
		"code":    code,
		"message": message,
	})
}
