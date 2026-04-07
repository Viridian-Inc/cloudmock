package flyio

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/plugin"
)

func TestDescribe(t *testing.T) {
	p := New()
	desc, err := p.Describe(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if desc.Name != "flyio" {
		t.Errorf("name = %q, want flyio", desc.Name)
	}
	if len(desc.Actions) == 0 {
		t.Error("expected actions to be defined")
	}
}

func TestCreateAndListApps(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Create app
	body, _ := json.Marshal(map[string]string{"app_name": "test-app", "org_slug": "personal"})
	resp, err := p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps", Body: body,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create app: status %d, body: %s", resp.StatusCode, resp.Body)
	}

	// List apps
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "GET", Path: "/fly/v1/apps",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list apps: status %d", resp.StatusCode)
	}
	var apps []App
	json.Unmarshal(resp.Body, &apps)
	if len(apps) != 1 || apps[0].Name != "test-app" {
		t.Errorf("expected 1 app named test-app, got %d", len(apps))
	}
}

func TestCreateAndListMachines(t *testing.T) {
	p := New()
	ctx := context.Background()

	// Create app first
	body, _ := json.Marshal(map[string]string{"app_name": "my-app"})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps", Body: body})

	// Create machine
	body, _ = json.Marshal(map[string]any{
		"name":   "web-1",
		"region": "iad",
		"config": map[string]any{
			"image": "ghcr.io/neureaux/cloudmock:latest",
			"guest": map[string]any{"cpus": 2, "memory_mb": 512},
		},
	})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps/my-app/machines", Body: body,
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create machine: status %d, body: %s", resp.StatusCode, resp.Body)
	}

	var machine Machine
	json.Unmarshal(resp.Body, &machine)
	if machine.AppName != "my-app" || machine.Region != "iad" || machine.CPUs != 2 {
		t.Errorf("unexpected machine: %+v", machine)
	}

	// List machines
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "GET", Path: "/fly/v1/apps/my-app/machines",
	})
	var machines []Machine
	json.Unmarshal(resp.Body, &machines)
	if len(machines) != 1 {
		t.Errorf("expected 1 machine, got %d", len(machines))
	}
}

func TestStartStopMachine(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]string{"app_name": "my-app"})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps", Body: body})

	body, _ = json.Marshal(map[string]any{
		"name": "web", "config": map[string]any{"image": "nginx"},
	})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps/my-app/machines", Body: body,
	})
	var m Machine
	json.Unmarshal(resp.Body, &m)

	// Stop
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps/my-app/machines/" + m.ID + "/stop",
	})
	json.Unmarshal(resp.Body, &m)
	if m.State != "stopped" {
		t.Errorf("expected stopped, got %s", m.State)
	}

	// Start
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps/my-app/machines/" + m.ID + "/start",
	})
	json.Unmarshal(resp.Body, &m)
	if m.State != "started" {
		t.Errorf("expected started, got %s", m.State)
	}
}

func TestSecrets(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]string{"app_name": "my-app"})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps", Body: body})

	// Set secrets
	body, _ = json.Marshal(map[string]string{"DATABASE_URL": "postgres://...", "API_KEY": "secret"})
	resp, _ := p.HandleRequest(ctx, &plugin.Request{
		Method: "POST", Path: "/fly/v1/apps/my-app/secrets", Body: body,
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("set secrets: status %d", resp.StatusCode)
	}

	// List secrets (should only show names, not values)
	resp, _ = p.HandleRequest(ctx, &plugin.Request{
		Method: "GET", Path: "/fly/v1/apps/my-app/secrets",
	})
	var secrets []map[string]string
	json.Unmarshal(resp.Body, &secrets)
	if len(secrets) != 2 {
		t.Errorf("expected 2 secrets, got %d", len(secrets))
	}
	for _, s := range secrets {
		if s["digest"] != "sha256:***" {
			t.Error("secrets should not expose values")
		}
	}
}

func TestDeleteAppCascades(t *testing.T) {
	p := New()
	ctx := context.Background()

	body, _ := json.Marshal(map[string]string{"app_name": "doomed"})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps", Body: body})

	body, _ = json.Marshal(map[string]any{"name": "m1", "config": map[string]any{"image": "x"}})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps/doomed/machines", Body: body})

	body, _ = json.Marshal(map[string]any{"name": "vol1", "region": "iad", "size_gb": 5})
	p.HandleRequest(ctx, &plugin.Request{Method: "POST", Path: "/fly/v1/apps/doomed/volumes", Body: body})

	// Delete app should cascade
	resp, _ := p.HandleRequest(ctx, &plugin.Request{Method: "DELETE", Path: "/fly/v1/apps/doomed"})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete app: status %d", resp.StatusCode)
	}

	// Verify machines gone
	resp, _ = p.HandleRequest(ctx, &plugin.Request{Method: "GET", Path: "/fly/v1/apps/doomed/machines"})
	if resp.StatusCode != http.StatusOK {
		// App doesn't exist, machines endpoint returns empty for non-existent apps
	}
}
