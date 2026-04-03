package snapshot

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/neureaux/cloudmock/pkg/routing"
	"github.com/neureaux/cloudmock/pkg/service"
)

// mockSnapshotService implements both service.Service and service.Snapshotable.
type mockSnapshotService struct {
	name  string
	state map[string]string
}

func (m *mockSnapshotService) Name() string                      { return m.name }
func (m *mockSnapshotService) Actions() []service.Action         { return nil }
func (m *mockSnapshotService) HealthCheck() error                { return nil }
func (m *mockSnapshotService) HandleRequest(_ *service.RequestContext) (*service.Response, error) {
	return nil, nil
}

func (m *mockSnapshotService) ExportState() (json.RawMessage, error) {
	return json.Marshal(m.state)
}

func (m *mockSnapshotService) ImportState(data json.RawMessage) error {
	m.state = make(map[string]string)
	return json.Unmarshal(data, &m.state)
}

// nonSnapshotService does not implement Snapshotable.
type nonSnapshotService struct{}

func (n *nonSnapshotService) Name() string                      { return "non-snap" }
func (n *nonSnapshotService) Actions() []service.Action         { return nil }
func (n *nonSnapshotService) HealthCheck() error                { return nil }
func (n *nonSnapshotService) HandleRequest(_ *service.RequestContext) (*service.Response, error) {
	return &service.Response{StatusCode: http.StatusOK}, nil
}

func TestExportImportRoundTrip(t *testing.T) {
	reg := routing.NewRegistry()

	orig := &mockSnapshotService{
		name:  "test-svc",
		state: map[string]string{"key1": "value1", "key2": "value2"},
	}
	reg.Register(orig)
	reg.Register(&nonSnapshotService{})

	// Export
	data, err := Export(reg)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify JSON structure
	var sf StateFile
	if err := json.Unmarshal(data, &sf); err != nil {
		t.Fatalf("Unmarshal StateFile: %v", err)
	}
	if sf.Version != 1 {
		t.Errorf("expected version 1, got %d", sf.Version)
	}
	if _, ok := sf.Services["test-svc"]; !ok {
		t.Fatal("expected test-svc in exported services")
	}
	if _, ok := sf.Services["non-snap"]; ok {
		t.Fatal("non-snapshotable service should not be exported")
	}

	// Create a fresh registry and import
	reg2 := routing.NewRegistry()
	target := &mockSnapshotService{name: "test-svc", state: make(map[string]string)}
	reg2.Register(target)

	if err := Import(reg2, data); err != nil {
		t.Fatalf("Import failed: %v", err)
	}

	if target.state["key1"] != "value1" {
		t.Errorf("expected key1=value1, got %q", target.state["key1"])
	}
	if target.state["key2"] != "value2" {
		t.Errorf("expected key2=value2, got %q", target.state["key2"])
	}
}

func TestImportSkipsUnknownServices(t *testing.T) {
	data := []byte(`{"version":1,"exported_at":"2026-01-01T00:00:00Z","services":{"unknown-svc":{"foo":"bar"}}}`)
	reg := routing.NewRegistry()
	// Should not error — just skip.
	if err := Import(reg, data); err != nil {
		t.Fatalf("Import should skip unknown services, got: %v", err)
	}
}

func TestImportInvalidJSON(t *testing.T) {
	if err := Import(routing.NewRegistry(), []byte("not-json")); err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}
