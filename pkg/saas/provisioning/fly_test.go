package provisioning

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestFly_CreateApp(t *testing.T) {
	var capturedBody flyCreateAppRequest
	var capturedAuth string

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")

		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/apps") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{}`))
	}))
	defer mock.Close()

	client := NewFlyClient("fly-test-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	client.httpClient = mock.Client()
	// Override the base URL by replacing the httpClient with a transport
	// that rewrites URLs.
	client.httpClient = &http.Client{
		Transport: &flyTestTransport{base: mock.URL, inner: http.DefaultTransport},
	}

	err := client.CreateApp(context.Background(), "cm-acme")
	if err != nil {
		t.Fatalf("CreateApp: %v", err)
	}

	if capturedBody.AppName != "cm-acme" {
		t.Errorf("AppName = %q, want %q", capturedBody.AppName, "cm-acme")
	}
	if capturedBody.OrgSlug != "my-org" {
		t.Errorf("OrgSlug = %q, want %q", capturedBody.OrgSlug, "my-org")
	}
	if capturedAuth != "Bearer fly-test-token" {
		t.Errorf("Authorization = %q, want %q", capturedAuth, "Bearer fly-test-token")
	}
}

func TestFly_CreateMachine(t *testing.T) {
	var capturedBody flyCreateMachineRequest

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if !strings.Contains(r.URL.Path, "/machines") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &capturedBody)

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(flyCreateMachineResponse{ID: "mach_12345"})
	}))
	defer mock.Close()

	client := NewFlyClient("fly-test-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	client.httpClient = &http.Client{
		Transport: &flyTestTransport{base: mock.URL, inner: http.DefaultTransport},
	}

	env := map[string]string{
		"CLOUDMOCK_TENANT_ID": "t-123",
		"CLOUDMOCK_AUTH":      "true",
	}

	machineID, err := client.CreateMachine(context.Background(), "cm-acme", env)
	if err != nil {
		t.Fatalf("CreateMachine: %v", err)
	}

	if machineID != "mach_12345" {
		t.Errorf("machineID = %q, want %q", machineID, "mach_12345")
	}
	if capturedBody.Region != "iad" {
		t.Errorf("Region = %q, want %q", capturedBody.Region, "iad")
	}
	if capturedBody.Config.Image != "registry.fly.io/cloudmock:latest" {
		t.Errorf("Image = %q, want %q", capturedBody.Config.Image, "registry.fly.io/cloudmock:latest")
	}
	if capturedBody.Config.Env["CLOUDMOCK_TENANT_ID"] != "t-123" {
		t.Errorf("Env[CLOUDMOCK_TENANT_ID] = %q, want %q", capturedBody.Config.Env["CLOUDMOCK_TENANT_ID"], "t-123")
	}
}

func TestFly_DestroyMachine(t *testing.T) {
	var capturedMethod string
	var capturedPath string

	mock := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedMethod = r.Method
		capturedPath = r.URL.Path

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer mock.Close()

	client := NewFlyClient("fly-test-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	client.httpClient = &http.Client{
		Transport: &flyTestTransport{base: mock.URL, inner: http.DefaultTransport},
	}

	err := client.DestroyMachine(context.Background(), "cm-acme", "mach_12345")
	if err != nil {
		t.Fatalf("DestroyMachine: %v", err)
	}

	if capturedMethod != http.MethodDelete {
		t.Errorf("Method = %q, want DELETE", capturedMethod)
	}
	if !strings.Contains(capturedPath, "mach_12345") {
		t.Errorf("Path = %q, expected to contain machine ID", capturedPath)
	}
	if !strings.Contains(capturedPath, "cm-acme") {
		t.Errorf("Path = %q, expected to contain app name", capturedPath)
	}
}

// flyTestTransport rewrites Fly API URLs to point to a test server.
type flyTestTransport struct {
	base  string
	inner http.RoundTripper
}

func (t *flyTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.URL.Scheme = "http"
	req.URL.Host = t.base[len("http://"):]
	return t.inner.RoundTrip(req)
}
