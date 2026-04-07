package route53_test

import (
	"encoding/json"
	"testing"

	r53svc "github.com/Viridian-Inc/cloudmock/services/route53"
)

const (
	r53TestAccount = "123456789012"
	r53TestRegion  = "us-east-1"
)

func TestRoute53_ExportState_Empty(t *testing.T) {
	svc := r53svc.New(r53TestAccount, r53TestRegion)

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}
	if !json.Valid(raw) {
		t.Fatalf("ExportState returned invalid JSON: %s", raw)
	}

	var state struct {
		HostedZones []any `json:"hosted_zones"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(state.HostedZones) != 0 {
		t.Errorf("expected empty hosted_zones, got %d", len(state.HostedZones))
	}
}

func TestRoute53_ExportState_WithHostedZones(t *testing.T) {
	svc := r53svc.New(r53TestAccount, r53TestRegion)

	seed := json.RawMessage(`{"hosted_zones":[{"name":"example.com.","comment":"public zone"},{"name":"internal.example.com.","comment":"private zone","private_zone":true}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		HostedZones []struct {
			Name        string `json:"name"`
			Comment     string `json:"comment"`
			PrivateZone bool   `json:"private_zone"`
		} `json:"hosted_zones"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.HostedZones) != 2 {
		t.Fatalf("expected 2 hosted zones, got %d", len(state.HostedZones))
	}

	names := make(map[string]bool)
	for _, z := range state.HostedZones {
		names[z.Name] = true
	}
	for _, expected := range []string{"example.com.", "internal.example.com."} {
		if !names[expected] {
			t.Errorf("zone %q not found in export", expected)
		}
	}
}

func TestRoute53_ExportState_PreservesComment(t *testing.T) {
	svc := r53svc.New(r53TestAccount, r53TestRegion)

	seed := json.RawMessage(`{"hosted_zones":[{"name":"myzone.io.","comment":"managed by terraform"}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	var state struct {
		HostedZones []struct {
			Name    string `json:"name"`
			Comment string `json:"comment"`
		} `json:"hosted_zones"`
	}
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if len(state.HostedZones) == 0 {
		t.Fatal("expected hosted zones in export")
	}
	if state.HostedZones[0].Comment != "managed by terraform" {
		t.Errorf("expected comment 'managed by terraform', got %q", state.HostedZones[0].Comment)
	}
}

func TestRoute53_ImportState_EmptyDoesNotCrash(t *testing.T) {
	svc := r53svc.New(r53TestAccount, r53TestRegion)

	if err := svc.ImportState(json.RawMessage(`{"hosted_zones":[]}`)); err != nil {
		t.Fatalf("ImportState with empty hosted_zones: %v", err)
	}
}

func TestRoute53_RoundTrip_PreservesZones(t *testing.T) {
	svc := r53svc.New(r53TestAccount, r53TestRegion)

	seed := json.RawMessage(`{"hosted_zones":[{"name":"alpha.com.","comment":"zone alpha"},{"name":"beta.org.","comment":"zone beta"}]}`)
	if err := svc.ImportState(seed); err != nil {
		t.Fatalf("ImportState: %v", err)
	}

	raw, err := svc.ExportState()
	if err != nil {
		t.Fatalf("ExportState: %v", err)
	}

	svc2 := r53svc.New(r53TestAccount, r53TestRegion)
	if err := svc2.ImportState(raw); err != nil {
		t.Fatalf("ImportState (svc2): %v", err)
	}

	raw2, err := svc2.ExportState()
	if err != nil {
		t.Fatalf("ExportState (svc2): %v", err)
	}

	var state struct {
		HostedZones []struct{ Name string `json:"name"` } `json:"hosted_zones"`
	}
	json.Unmarshal(raw2, &state)

	names := make(map[string]bool)
	for _, z := range state.HostedZones {
		names[z.Name] = true
	}
	for _, expected := range []string{"alpha.com.", "beta.org."} {
		if !names[expected] {
			t.Errorf("zone %q not restored", expected)
		}
	}
}
