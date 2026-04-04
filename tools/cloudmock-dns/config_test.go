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
    autotend: custom.example.com
    cloudmock: mock.example.com
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dc.Autotend != "custom.example.com" {
		t.Errorf("expected custom.example.com, got %s", dc.Autotend)
	}
	if dc.Cloudmock != "mock.example.com" {
		t.Errorf("expected mock.example.com, got %s", dc.Cloudmock)
	}
}

func TestParsePulumiConfigMissing(t *testing.T) {
	dc, err := parsePulumiConfig("/nonexistent/path.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
	if dc.Autotend != "cloudmock.app" || dc.Cloudmock != "cloudmock.app" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}

func TestParsePulumiConfigNoDomains(t *testing.T) {
	yaml := `config:
  autotend-backend:environment: local
`
	tmp := filepath.Join(t.TempDir(), "Pulumi.local.yaml")
	os.WriteFile(tmp, []byte(yaml), 0644)

	dc, err := parsePulumiConfig(tmp)
	if err == nil {
		t.Error("expected error when domains key is missing")
	}
	if dc.Autotend != "cloudmock.app" || dc.Cloudmock != "cloudmock.app" {
		t.Errorf("expected default domains, got %+v", dc)
	}
}
