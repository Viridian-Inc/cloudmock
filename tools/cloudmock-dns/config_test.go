package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePulumiConfig(t *testing.T) {
	yaml := `config:
  backend:environment: local
  backend:domains:
    primary: custom.example.com
    cloudmock: mock.example.com
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Primary != "custom.example.com" {
		t.Errorf("expected custom.example.com, got %s", dc.Primary)
	}
	if dc.Cloudmock != "mock.example.com" {
		t.Errorf("expected mock.example.com, got %s", dc.Cloudmock)
	}
}

func TestParsePulumiConfig_LegacyAutotendKey(t *testing.T) {
	// Existing autotend-infra Pulumi configs use the "autotend" key for the
	// primary domain. Verify the legacy alias still maps to dc.Primary.
	yaml := `config:
  backend:environment: local
  backend:domains:
    autotend: legacy.example.com
    cloudmock: mock.example.com
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Primary != "legacy.example.com" {
		t.Errorf("expected legacy.example.com from autotend alias, got %s", dc.Primary)
	}
}

func TestParsePulumiConfig_PrimaryWinsOverLegacy(t *testing.T) {
	// If both keys are set, "primary" wins.
	yaml := `config:
  backend:domains:
    primary: new.example.com
    autotend: old.example.com
    cloudmock: mock.example.com
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, _ := parsePulumiConfig(tmp)
	if dc.Primary != "new.example.com" {
		t.Errorf("expected primary key to win, got %s", dc.Primary)
	}
}

func TestParsePulumiConfigMissing(t *testing.T) {
	dc, err := parsePulumiConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if dc.Primary != "cloudmock.app" || dc.Cloudmock != "cloudmock.app" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}

func TestParsePulumiConfigNoDomains(t *testing.T) {
	yaml := `config:
  backend:environment: local
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err == nil {
		t.Error("expected error when domains key is missing")
	}
	if dc.Primary != "cloudmock.app" || dc.Cloudmock != "cloudmock.app" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}
