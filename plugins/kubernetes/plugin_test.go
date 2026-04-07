package kubernetes

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/plugin"
	"github.com/Viridian-Inc/cloudmock/plugins/kubernetes/internal"
)

func TestDescribe(t *testing.T) {
	p := New()
	desc, err := p.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if desc.Name != "kubernetes" {
		t.Errorf("name = %q, want kubernetes", desc.Name)
	}
	if desc.Protocol != "k8s-api" {
		t.Errorf("protocol = %q, want k8s-api", desc.Protocol)
	}
}

func TestHealthCheck(t *testing.T) {
	p := New()
	status, _, err := p.HealthCheck(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if status != plugin.HealthHealthy {
		t.Errorf("status = %v, want HealthHealthy", status)
	}
}

func TestDiscoveryAPI(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, err := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("/api status = %d, want 200", resp.StatusCode)
	}
	var versions map[string]interface{}
	json.Unmarshal(resp.Body, &versions)
	if versions["kind"] != "APIVersions" {
		t.Errorf("/api kind = %v, want APIVersions", versions["kind"])
	}
}

func TestDiscoveryAPIs(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, err := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/apis"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("/apis status = %d, want 200", resp.StatusCode)
	}
	var groups map[string]interface{}
	json.Unmarshal(resp.Body, &groups)
	if groups["kind"] != "APIGroupList" {
		t.Errorf("/apis kind = %v, want APIGroupList", groups["kind"])
	}
}

func TestVersion(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, err := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/version"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("/version status = %d, want 200", resp.StatusCode)
	}
}

func TestListDefaultNamespaces(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, err := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, body = %s", resp.StatusCode, resp.Body)
	}
	var list internal.NamespaceList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 4 {
		t.Errorf("expected 4 default namespaces, got %d", len(list.Items))
	}
}

func TestCreateAndGetNamespace(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "test-ns"},
	})

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/namespaces", Body: body})
	if resp.StatusCode != 201 {
		t.Fatalf("create status = %d, body = %s", resp.StatusCode, resp.Body)
	}

	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/test-ns"})
	if resp.StatusCode != 200 {
		t.Fatalf("get status = %d", resp.StatusCode)
	}
	var ns internal.Namespace
	json.Unmarshal(resp.Body, &ns)
	if ns.Name != "test-ns" {
		t.Errorf("name = %q, want test-ns", ns.Name)
	}
}

func TestPodCRUD(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Create pod
	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{"name": "nginx", "labels": map[string]string{"app": "nginx"}},
		"spec": map[string]interface{}{
			"containers": []map[string]string{{"name": "nginx", "image": "nginx:latest"}},
		},
	})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/namespaces/default/pods", Body: body})
	if resp.StatusCode != 201 {
		t.Fatalf("create pod status = %d, body = %s", resp.StatusCode, resp.Body)
	}

	// Get pod
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/default/pods/nginx"})
	if resp.StatusCode != 200 {
		t.Fatalf("get pod status = %d", resp.StatusCode)
	}
	var pod internal.Pod
	json.Unmarshal(resp.Body, &pod)
	if pod.Status.Phase != "Running" {
		t.Errorf("pod phase = %q, want Running", pod.Status.Phase)
	}
	if pod.Spec.NodeName != "cloudmock-node" {
		t.Errorf("nodeName = %q, want cloudmock-node", pod.Spec.NodeName)
	}

	// List pods
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/default/pods"})
	if resp.StatusCode != 200 {
		t.Fatal("list pods failed")
	}
	var podList internal.PodList
	json.Unmarshal(resp.Body, &podList)
	if len(podList.Items) != 1 {
		t.Errorf("pod count = %d, want 1", len(podList.Items))
	}

	// List with label selector
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "GET", Path: "/api/v1/namespaces/default/pods",
		QueryParams: map[string]string{"labelSelector": "app=nginx"},
	})
	json.Unmarshal(resp.Body, &podList)
	if len(podList.Items) != 1 {
		t.Errorf("filtered pod count = %d, want 1", len(podList.Items))
	}

	// Delete pod
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "DELETE", Path: "/api/v1/namespaces/default/pods/nginx"})
	if resp.StatusCode != 200 {
		t.Fatalf("delete pod status = %d", resp.StatusCode)
	}

	// Verify deleted
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/default/pods/nginx"})
	if resp.StatusCode != 404 {
		t.Errorf("get deleted pod status = %d, want 404", resp.StatusCode)
	}
}

func TestDeploymentCRUD(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "web"},
		"spec": map[string]interface{}{
			"replicas": 3,
			"selector": map[string]interface{}{"matchLabels": map[string]string{"app": "web"}},
			"template": map[string]interface{}{
				"spec": map[string]interface{}{
					"containers": []map[string]string{{"name": "web", "image": "nginx:1.25"}},
				},
			},
		},
	})

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/apis/apps/v1/namespaces/default/deployments", Body: body})
	if resp.StatusCode != 201 {
		t.Fatalf("create deployment status = %d, body = %s", resp.StatusCode, resp.Body)
	}

	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/apis/apps/v1/namespaces/default/deployments/web"})
	if resp.StatusCode != 200 {
		t.Fatalf("get deployment status = %d", resp.StatusCode)
	}
	var dep internal.Deployment
	json.Unmarshal(resp.Body, &dep)
	if dep.Status.ReadyReplicas != 3 {
		t.Errorf("readyReplicas = %d, want 3", dep.Status.ReadyReplicas)
	}
}

func TestServiceCRUD(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]interface{}{
		"metadata": map[string]string{"name": "my-svc"},
		"spec": map[string]interface{}{
			"selector": map[string]string{"app": "web"},
			"ports":    []map[string]interface{}{{"port": 80, "targetPort": 8080}},
		},
	})

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/api/v1/namespaces/default/services", Body: body})
	if resp.StatusCode != 201 {
		t.Fatalf("create service status = %d, body = %s", resp.StatusCode, resp.Body)
	}

	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/namespaces/default/services/my-svc"})
	if resp.StatusCode != 200 {
		t.Fatalf("get service status = %d", resp.StatusCode)
	}
	var svc internal.Service
	json.Unmarshal(resp.Body, &svc)
	if svc.Spec.Type != "ClusterIP" {
		t.Errorf("type = %q, want ClusterIP", svc.Spec.Type)
	}
}

func TestListNodes(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/api/v1/nodes"})
	if resp.StatusCode != 200 {
		t.Fatalf("list nodes status = %d", resp.StatusCode)
	}
	var list internal.NodeList
	json.Unmarshal(resp.Body, &list)
	if len(list.Items) != 1 {
		t.Errorf("node count = %d, want 1", len(list.Items))
	}
	if list.Items[0].Name != "cloudmock-node" {
		t.Errorf("node name = %q, want cloudmock-node", list.Items[0].Name)
	}
}

func TestNotFoundPath(t *testing.T) {
	p := New()
	ctx := context.Background()

	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/nonexistent"})
	if resp.StatusCode != 404 {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
