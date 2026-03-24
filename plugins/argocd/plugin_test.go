package argocd

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/neureaux/cloudmock/pkg/plugin"
	"github.com/neureaux/cloudmock/plugins/argocd/internal"
	k8splugin "github.com/neureaux/cloudmock/plugins/kubernetes"
)

func TestDescribe(t *testing.T) {
	p := New(nil)
	desc, err := p.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if desc.Name != "argocd" {
		t.Errorf("name = %q, want argocd", desc.Name)
	}
}

func TestSession(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/session", Body: []byte(`{"username":"admin","password":"cloudmock"}`)})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var session internal.Session
	json.Unmarshal(resp.Body, &session)
	if session.Token == "" {
		t.Error("expected non-empty token")
	}
}

func TestSettings(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/settings"})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestApplicationCRUD(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	// Create
	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "myapp"},
		"spec": map[string]interface{}{
			"source":      map[string]string{"repoURL": "https://github.com/org/repo", "path": "k8s/"},
			"destination": map[string]string{"server": "https://kubernetes.default.svc", "namespace": "production"},
			"project":     "default",
		},
	})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/applications", Body: body})
	if resp.StatusCode != 200 {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, resp.Body)
	}

	// Get
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/applications/myapp"})
	if resp.StatusCode != 200 {
		t.Fatalf("get status = %d", resp.StatusCode)
	}
	var app internal.Application
	json.Unmarshal(resp.Body, &app)
	if app.Status.Sync.Status != "OutOfSync" {
		t.Errorf("sync status = %q, want OutOfSync", app.Status.Sync.Status)
	}

	// List
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/applications"})
	if resp.StatusCode != 200 {
		t.Fatal("list failed")
	}
	var list internal.ApplicationList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 1 {
		t.Errorf("app count = %d, want 1", len(list.Items))
	}

	// Delete
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "DELETE", Path: "/api/v1/applications/myapp"})
	if resp.StatusCode != 200 {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}

	// Verify deleted
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/applications/myapp"})
	if resp.StatusCode != 404 {
		t.Errorf("get deleted app status = %d, want 404", resp.StatusCode)
	}
}

func TestSync(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	// Create app
	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "web"},
		"spec": map[string]interface{}{
			"source":      map[string]string{"repoURL": "https://github.com/org/repo", "path": "k8s/"},
			"destination": map[string]string{"namespace": "staging"},
		},
	})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/applications", Body: body})

	// Sync
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/applications/web/sync"})
	if resp.StatusCode != 200 {
		t.Fatalf("sync status = %d, body = %s", resp.StatusCode, resp.Body)
	}
	var app internal.Application
	json.Unmarshal(resp.Body, &app)
	if app.Status.Sync.Status != "Synced" {
		t.Errorf("sync status = %q, want Synced", app.Status.Sync.Status)
	}
	if app.Status.Health.Status != "Healthy" {
		t.Errorf("health = %q, want Healthy", app.Status.Health.Status)
	}
	if app.Status.OperationState == nil || app.Status.OperationState.Phase != "Succeeded" {
		t.Error("expected operation phase Succeeded")
	}
}

func TestSyncWithK8sIntegration(t *testing.T) {
	k8s := k8splugin.New()
	p := New(k8s)
	ctx := context.Background()

	// Create app targeting "production" namespace
	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "backend"},
		"spec": map[string]interface{}{
			"source":      map[string]string{"repoURL": "https://github.com/org/repo", "path": "deploy/"},
			"destination": map[string]string{"namespace": "production"},
		},
	})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/applications", Body: body})

	// Sync — should create namespace + deployment in k8s plugin
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/applications/backend/sync"})
	if resp.StatusCode != 200 {
		t.Fatalf("sync status = %d", resp.StatusCode)
	}

	// Verify k8s plugin has the namespace
	k8sResp, _ := k8s.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/production"})
	if k8sResp.StatusCode != 200 {
		t.Fatalf("k8s namespace status = %d, body = %s", k8sResp.StatusCode, k8sResp.Body)
	}

	// Verify k8s plugin has the deployment
	k8sResp, _ = k8s.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/apis/apps/v1/namespaces/production/deployments/backend"})
	if k8sResp.StatusCode != 200 {
		t.Fatalf("k8s deployment status = %d, body = %s", k8sResp.StatusCode, k8sResp.Body)
	}
}

func TestRepositoryCRUD(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	body, _ := json.Marshal(map[string]interface{}{
		"repo": "https://github.com/org/repo",
		"name": "my-repo",
	})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/repositories", Body: body})
	if resp.StatusCode != 200 {
		t.Fatalf("create repo status = %d", resp.StatusCode)
	}

	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/repositories"})
	var list internal.RepositoryList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 1 {
		t.Errorf("repo count = %d, want 1", len(list.Items))
	}
}

func TestClusters(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/clusters"})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var list internal.ClusterList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 1 {
		t.Errorf("cluster count = %d, want 1 (in-cluster)", len(list.Items))
	}
}

func TestProjects(t *testing.T) {
	p := New(nil)
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/projects"})
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
	var list internal.AppProjectList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 1 {
		t.Errorf("project count = %d, want 1 (default)", len(list.Items))
	}
}
