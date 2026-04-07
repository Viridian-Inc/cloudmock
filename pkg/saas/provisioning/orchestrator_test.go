package provisioning

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/saas/tenant"
)

// mockFlyServer creates a Fly mock that handles app and machine operations.
// It records requests for later assertion.
type mockFlyState struct {
	mu       sync.Mutex
	apps     map[string]bool
	machines map[string]string // machineID -> appName
	requests []mockHTTPRequest
}

type mockHTTPRequest struct {
	Method string
	Path   string
}

func newMockFly(t *testing.T) (*httptest.Server, *mockFlyState) {
	t.Helper()
	state := &mockFlyState{
		apps:     make(map[string]bool),
		machines: make(map[string]string),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.requests = append(state.requests, mockHTTPRequest{Method: r.Method, Path: r.URL.Path})
		state.mu.Unlock()

		path := r.URL.Path

		// POST /v1/apps — create app
		if r.Method == http.MethodPost && strings.HasSuffix(path, "/apps") {
			body, _ := io.ReadAll(r.Body)
			var req flyCreateAppRequest
			json.Unmarshal(body, &req)
			state.mu.Lock()
			state.apps[req.AppName] = true
			state.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{}`))
			return
		}

		// POST /v1/apps/{app}/machines — create machine
		if r.Method == http.MethodPost && strings.Contains(path, "/machines") && !strings.Contains(path, "/stop") {
			machineID := fmt.Sprintf("mach_%s", path)
			state.mu.Lock()
			// Extract app name from path.
			parts := strings.Split(path, "/")
			for i, p := range parts {
				if p == "apps" && i+1 < len(parts) {
					state.machines[machineID] = parts[i+1]
					break
				}
			}
			state.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(flyCreateMachineResponse{ID: machineID})
			return
		}

		// DELETE /v1/apps/{app}/machines/{id} — destroy machine
		if r.Method == http.MethodDelete && strings.Contains(path, "/machines/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}

		// DELETE /v1/apps/{app} — delete app
		if r.Method == http.MethodDelete && strings.Contains(path, "/apps/") {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`))
			return
		}

		t.Logf("mock fly: unhandled request %s %s", r.Method, path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv, state
}

// mockCloudflareServer creates a Cloudflare mock.
type mockCFState struct {
	mu       sync.Mutex
	records  map[string]string // name -> recordID
	requests []mockHTTPRequest
	failNext bool
}

func newMockCloudflare(t *testing.T) (*httptest.Server, *mockCFState) {
	t.Helper()
	state := &mockCFState{
		records: make(map[string]string),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		state.mu.Lock()
		state.requests = append(state.requests, mockHTTPRequest{Method: r.Method, Path: r.URL.Path})
		shouldFail := state.failNext
		state.mu.Unlock()

		if shouldFail {
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(cfAPIResponse{
				Success: false,
				Errors:  []cfAPIError{{Code: 500, Message: "injected failure"}},
			})
			return
		}

		// POST dns_records — add record
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/dns_records") {
			body, _ := io.ReadAll(r.Body)
			var req cfCreateRecordRequest
			json.Unmarshal(body, &req)
			recordID := "rec_" + req.Name
			state.mu.Lock()
			state.records[req.Name] = recordID
			state.mu.Unlock()
			w.WriteHeader(http.StatusOK)
			resp := cfAPIResponse{
				Success: true,
				Result:  json.RawMessage(fmt.Sprintf(`{"id":%q}`, recordID)),
			}
			json.NewEncoder(w).Encode(resp)
			return
		}

		// GET dns_records — find record
		if r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/dns_records") {
			name := r.URL.Query().Get("name")
			state.mu.Lock()
			recordID, ok := state.records[name]
			state.mu.Unlock()
			if !ok {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(cfAPIResponse{
					Success: true,
					Result:  json.RawMessage(`[]`),
				})
				return
			}
			records := []cfDNSRecord{{ID: recordID, Name: name, Type: "CNAME"}}
			recordsJSON, _ := json.Marshal(records)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cfAPIResponse{Success: true, Result: recordsJSON})
			return
		}

		// DELETE dns_records/{id} — delete record
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/dns_records/") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(cfAPIResponse{
				Success: true,
				Result:  json.RawMessage(`{"id":"deleted"}`),
			})
			return
		}

		t.Logf("mock cf: unhandled request %s %s", r.Method, r.URL.Path)
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(srv.Close)
	return srv, state
}

func TestOrchestrator_Provision(t *testing.T) {
	flySrv, flyState := newMockFly(t)
	cfSrv, _ := newMockCloudflare(t)
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID: "org_prov_1",
		Name:       "Provision Test",
		Slug:       "prov-test",
		Tier:       "pro",
		Status:     "active",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	flyClient := NewFlyClient("fly-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	flyClient.httpClient = &http.Client{
		Transport: &flyTestTransport{base: flySrv.URL, inner: http.DefaultTransport},
	}

	cfClient := NewCloudflareClient("cf-token", "zone_abc")
	cfClient.httpClient = &http.Client{
		Transport: &cfTestTransport{base: cfSrv.URL, inner: http.DefaultTransport},
	}

	orch := NewOrchestrator(flyClient, cfClient, store)
	if err := orch.Provision(ctx, tn); err != nil {
		t.Fatalf("Provision: %v", err)
	}

	// Verify Fly app was created.
	flyState.mu.Lock()
	if !flyState.apps["cm-prov-test"] {
		t.Error("Fly app 'cm-prov-test' was not created")
	}
	flyState.mu.Unlock()

	// Verify tenant was updated with machine info.
	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FlyAppName != "cm-prov-test" {
		t.Errorf("FlyAppName = %q, want %q", got.FlyAppName, "cm-prov-test")
	}
	if got.FlyMachineID == "" {
		t.Error("FlyMachineID should be set after provisioning")
	}
}

func TestOrchestrator_Deprovision(t *testing.T) {
	flySrv, _ := newMockFly(t)
	cfSrv, cfState := newMockCloudflare(t)
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	tn := &tenant.Tenant{
		ClerkOrgID:   "org_deprov_1",
		Name:         "Deprovision Test",
		Slug:         "deprov-test",
		Tier:         "pro",
		Status:       "active",
		FlyAppName:   "cm-deprov-test",
		FlyMachineID: "mach_xyz",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	// Pre-populate a DNS record so RemoveCNAME can find it.
	cfState.mu.Lock()
	cfState.records["deprov-test.cloudmock.app"] = "rec_deprov"
	cfState.mu.Unlock()

	flyClient := NewFlyClient("fly-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	flyClient.httpClient = &http.Client{
		Transport: &flyTestTransport{base: flySrv.URL, inner: http.DefaultTransport},
	}

	cfClient := NewCloudflareClient("cf-token", "zone_abc")
	cfClient.httpClient = &http.Client{
		Transport: &cfTestTransport{base: cfSrv.URL, inner: http.DefaultTransport},
	}

	orch := NewOrchestrator(flyClient, cfClient, store)
	if err := orch.Deprovision(ctx, tn); err != nil {
		t.Fatalf("Deprovision: %v", err)
	}

	// Verify tenant fields were cleared.
	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FlyAppName != "" {
		t.Errorf("FlyAppName = %q, want empty", got.FlyAppName)
	}
	if got.FlyMachineID != "" {
		t.Errorf("FlyMachineID = %q, want empty", got.FlyMachineID)
	}
}

func TestOrchestrator_PartialFailure(t *testing.T) {
	// Scenario: Fly app and machine creation succeed, but Cloudflare DNS fails.
	// The orchestrator should clean up Fly (destroy machine + delete app).
	flySrv, flyState := newMockFly(t)
	cfSrv, cfState := newMockCloudflare(t)
	store := tenant.NewMemoryStore()
	ctx := context.Background()

	// Make Cloudflare fail.
	cfState.mu.Lock()
	cfState.failNext = true
	cfState.mu.Unlock()

	tn := &tenant.Tenant{
		ClerkOrgID: "org_partial_1",
		Name:       "Partial Fail Test",
		Slug:       "partial-fail",
		Tier:       "pro",
		Status:     "active",
	}
	if err := store.Create(ctx, tn); err != nil {
		t.Fatalf("Create tenant: %v", err)
	}

	flyClient := NewFlyClient("fly-token", "my-org", "iad", "registry.fly.io/cloudmock:latest")
	flyClient.httpClient = &http.Client{
		Transport: &flyTestTransport{base: flySrv.URL, inner: http.DefaultTransport},
	}

	cfClient := NewCloudflareClient("cf-token", "zone_abc")
	cfClient.httpClient = &http.Client{
		Transport: &cfTestTransport{base: cfSrv.URL, inner: http.DefaultTransport},
	}

	orch := NewOrchestrator(flyClient, cfClient, store)
	err := orch.Provision(ctx, tn)
	if err == nil {
		t.Fatal("expected error from Provision when Cloudflare fails, got nil")
	}

	// Verify that cleanup happened: there should be DELETE requests to Fly
	// for both the machine and the app.
	flyState.mu.Lock()
	var flyDeletes int
	for _, req := range flyState.requests {
		if req.Method == http.MethodDelete {
			flyDeletes++
		}
	}
	flyState.mu.Unlock()

	// We expect at least 2 DELETE requests: one for the machine, one for the app.
	if flyDeletes < 2 {
		t.Errorf("expected at least 2 Fly DELETE requests for cleanup, got %d", flyDeletes)
	}

	// Tenant should NOT have been updated with Fly info since provisioning failed.
	got, err := store.Get(ctx, tn.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.FlyAppName != "" {
		t.Errorf("FlyAppName = %q, want empty after failed provision", got.FlyAppName)
	}
}
