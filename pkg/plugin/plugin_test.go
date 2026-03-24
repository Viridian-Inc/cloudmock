package plugin

import (
	"context"
	"encoding/json"
	"log/slog"
	"testing"
)

// testPlugin is a minimal plugin for unit testing.
type testPlugin struct {
	name string
}

func (p *testPlugin) Init(_ context.Context, _ []byte, _ string, _ string) error { return nil }
func (p *testPlugin) Shutdown(_ context.Context) error                           { return nil }
func (p *testPlugin) HealthCheck(_ context.Context) (HealthStatus, string, error) {
	return HealthHealthy, "", nil
}
func (p *testPlugin) Describe(_ context.Context) (*Descriptor, error) {
	return &Descriptor{
		Name:     p.name,
		Version:  "1.0.0",
		Protocol: "test",
		Actions:  []string{"TestAction"},
		APIPaths: []string{"/test/" + p.name + "/*"},
	}, nil
}
func (p *testPlugin) HandleRequest(_ context.Context, req *Request) (*Response, error) {
	body, _ := json.Marshal(map[string]string{"plugin": p.name, "action": req.Action})
	return &Response{StatusCode: 200, Body: body, Headers: map[string]string{"Content-Type": "application/json"}}, nil
}

func TestManagerRegisterAndLookup(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	p := &testPlugin{name: "test-svc"}
	if err := mgr.RegisterInProcess(ctx, p); err != nil {
		t.Fatalf("RegisterInProcess: %v", err)
	}

	got, err := mgr.Lookup("test-svc")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if got != p {
		t.Fatal("Lookup returned wrong plugin")
	}
}

func TestManagerDuplicateRegister(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	p := &testPlugin{name: "dup"}
	if err := mgr.RegisterInProcess(ctx, p); err != nil {
		t.Fatalf("first register: %v", err)
	}
	if err := mgr.RegisterInProcess(ctx, &testPlugin{name: "dup"}); err == nil {
		t.Fatal("expected error on duplicate register")
	}
}

func TestManagerLookupByPath(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	k8s := &testPlugin{name: "kubernetes"}
	// Override with custom paths for this test.
	mgr.plugins["kubernetes"] = &managedPlugin{
		plugin: k8s,
		descriptor: &Descriptor{
			Name:     "kubernetes",
			APIPaths: []string{"/api/v1/*", "/apis/*"},
		},
		mode:    ModeInProcess,
		healthy: true,
	}

	argo := &testPlugin{name: "argocd"}
	mgr.plugins["argocd"] = &managedPlugin{
		plugin: argo,
		descriptor: &Descriptor{
			Name:     "argocd",
			APIPaths: []string{"/api/v1/applications/*", "/api/v1/repositories/*"},
		},
		mode:    ModeInProcess,
		healthy: true,
	}

	tests := []struct {
		path     string
		wantName string
	}{
		{"/api/v1/pods", "kubernetes"},
		{"/apis/apps/v1/deployments", "kubernetes"},
		{"/api/v1/applications/myapp", "argocd"}, // more specific ArgoCD path wins
		{"/api/v1/repositories/myrepo", "argocd"},
		{"/unmatched/path", ""},
	}

	for _, tt := range tests {
		p, _ := mgr.LookupByPath(tt.path)
		if tt.wantName == "" {
			if p != nil {
				t.Errorf("LookupByPath(%q) = %v, want nil", tt.path, p)
			}
			continue
		}
		if p == nil {
			t.Errorf("LookupByPath(%q) = nil, want %q", tt.path, tt.wantName)
			continue
		}
		desc, _ := p.Describe(ctx)
		if desc.Name != tt.wantName {
			t.Errorf("LookupByPath(%q) = %q, want %q", tt.path, desc.Name, tt.wantName)
		}
	}
}

func TestManagerList(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	mgr.RegisterInProcess(ctx, &testPlugin{name: "a"})
	mgr.RegisterInProcess(ctx, &testPlugin{name: "b"})

	list := mgr.List()
	if len(list) != 2 {
		t.Fatalf("List() returned %d plugins, want 2", len(list))
	}

	names := map[string]bool{}
	for _, info := range list {
		names[info.Name] = true
	}
	if !names["a"] || !names["b"] {
		t.Errorf("List() names = %v, want a and b", names)
	}
}

func TestManagerHealthCheckAll(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	mgr.RegisterInProcess(ctx, &testPlugin{name: "healthy-svc"})

	results := mgr.HealthCheckAll(ctx)
	if len(results) != 1 {
		t.Fatalf("HealthCheckAll returned %d results, want 1", len(results))
	}
	r := results["healthy-svc"]
	if r.Status != HealthHealthy {
		t.Errorf("health status = %v, want HealthHealthy", r.Status)
	}
}

func TestManagerHandleRequest(t *testing.T) {
	mgr := NewManager(slog.Default())
	ctx := context.Background()

	mgr.RegisterInProcess(ctx, &testPlugin{name: "echo"})

	p, _ := mgr.Lookup("echo")
	resp, err := p.HandleRequest(ctx, &Request{Action: "TestAction"})
	if err != nil {
		t.Fatalf("HandleRequest: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	var body map[string]string
	json.Unmarshal(resp.Body, &body)
	if body["plugin"] != "echo" {
		t.Errorf("body.plugin = %q, want %q", body["plugin"], "echo")
	}
}
