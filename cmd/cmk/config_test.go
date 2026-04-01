package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultCmkConfig(t *testing.T) {
	cfg := DefaultCmkConfig()

	if cfg.Gateway.Port != 4566 {
		t.Errorf("expected gateway port 4566, got %d", cfg.Gateway.Port)
	}
	if cfg.Admin.Port != 4599 {
		t.Errorf("expected admin port 4599, got %d", cfg.Admin.Port)
	}
	if cfg.Dashboard.Port != 4500 {
		t.Errorf("expected dashboard port 4500, got %d", cfg.Dashboard.Port)
	}
	if !cfg.Dashboard.Enabled {
		t.Error("expected dashboard enabled by default")
	}
	if cfg.OTLP.Port != 4318 {
		t.Errorf("expected OTLP port 4318, got %d", cfg.OTLP.Port)
	}
	if cfg.Profile != "standard" {
		t.Errorf("expected profile 'standard', got %q", cfg.Profile)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected region 'us-east-1', got %q", cfg.Region)
	}
	if cfg.AccountID != "000000000000" {
		t.Errorf("expected account ID '000000000000', got %q", cfg.AccountID)
	}
}

func TestLoadCmkConfig(t *testing.T) {
	content := `
gateway:
  port: 5566
admin:
  port: 5599
dashboard:
  port: 5500
  enabled: false
otlp:
  port: 5318
iac:
  path: ./my-infra
  env: staging
profile: full
region: eu-west-1
account_id: "123456789012"
`
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".cloudmock.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadCmkConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Gateway.Port != 5566 {
		t.Errorf("expected gateway port 5566, got %d", cfg.Gateway.Port)
	}
	if cfg.Admin.Port != 5599 {
		t.Errorf("expected admin port 5599, got %d", cfg.Admin.Port)
	}
	if cfg.Dashboard.Port != 5500 {
		t.Errorf("expected dashboard port 5500, got %d", cfg.Dashboard.Port)
	}
	if cfg.Dashboard.Enabled {
		t.Error("expected dashboard disabled")
	}
	if cfg.OTLP.Port != 5318 {
		t.Errorf("expected OTLP port 5318, got %d", cfg.OTLP.Port)
	}
	if cfg.IaC.Path != "./my-infra" {
		t.Errorf("expected IaC path './my-infra', got %q", cfg.IaC.Path)
	}
	if cfg.IaC.Env != "staging" {
		t.Errorf("expected IaC env 'staging', got %q", cfg.IaC.Env)
	}
	if cfg.Profile != "full" {
		t.Errorf("expected profile 'full', got %q", cfg.Profile)
	}
	if cfg.Region != "eu-west-1" {
		t.Errorf("expected region 'eu-west-1', got %q", cfg.Region)
	}
	if cfg.AccountID != "123456789012" {
		t.Errorf("expected account ID '123456789012', got %q", cfg.AccountID)
	}
}

func TestLoadCmkConfig_PartialOverride(t *testing.T) {
	// Only override some fields; defaults should fill the rest.
	content := `
gateway:
  port: 9999
profile: minimal
`
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".cloudmock.yaml")
	if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadCmkConfig(cfgPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Gateway.Port != 9999 {
		t.Errorf("expected gateway port 9999, got %d", cfg.Gateway.Port)
	}
	if cfg.Profile != "minimal" {
		t.Errorf("expected profile 'minimal', got %q", cfg.Profile)
	}
	// Defaults should be preserved
	if cfg.Admin.Port != 4599 {
		t.Errorf("expected default admin port 4599, got %d", cfg.Admin.Port)
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("expected default region 'us-east-1', got %q", cfg.Region)
	}
}

func TestLoadCmkConfig_NotFound(t *testing.T) {
	_, err := LoadCmkConfig("/nonexistent/path/.cloudmock.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoadCmkConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, ".cloudmock.yaml")
	if err := os.WriteFile(cfgPath, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadCmkConfig(cfgPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestFindConfigFile(t *testing.T) {
	// Create a nested directory structure with a config file at the root
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}

	cfgPath := filepath.Join(root, ".cloudmock.yaml")
	if err := os.WriteFile(cfgPath, []byte("profile: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Should find config from nested directory
	found := FindConfigFile(nested)
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}

	// Should find config from root itself
	found = FindConfigFile(root)
	if found != cfgPath {
		t.Errorf("expected %q, got %q", cfgPath, found)
	}
}

func TestFindConfigFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	found := FindConfigFile(dir)
	if found != "" {
		t.Errorf("expected empty string, got %q", found)
	}
}

func TestApplyEnvOverrides(t *testing.T) {
	cfg := DefaultCmkConfig()

	t.Setenv("CLOUDMOCK_GATEWAY_PORT", "7777")
	t.Setenv("CLOUDMOCK_ADMIN_PORT", "7799")
	t.Setenv("CLOUDMOCK_DASHBOARD_PORT", "7700")
	t.Setenv("CLOUDMOCK_REGION", "ap-southeast-1")
	t.Setenv("CLOUDMOCK_PROFILE", "full")

	cfg.ApplyEnvOverrides()

	if cfg.Gateway.Port != 7777 {
		t.Errorf("expected gateway port 7777, got %d", cfg.Gateway.Port)
	}
	if cfg.Admin.Port != 7799 {
		t.Errorf("expected admin port 7799, got %d", cfg.Admin.Port)
	}
	if cfg.Dashboard.Port != 7700 {
		t.Errorf("expected dashboard port 7700, got %d", cfg.Dashboard.Port)
	}
	if cfg.Region != "ap-southeast-1" {
		t.Errorf("expected region 'ap-southeast-1', got %q", cfg.Region)
	}
	if cfg.Profile != "full" {
		t.Errorf("expected profile 'full', got %q", cfg.Profile)
	}
}

func TestAdminAddr(t *testing.T) {
	cfg := DefaultCmkConfig()
	if cfg.AdminAddr() != "http://localhost:4599" {
		t.Errorf("expected 'http://localhost:4599', got %q", cfg.AdminAddr())
	}

	cfg.Admin.Port = 9999
	if cfg.AdminAddr() != "http://localhost:9999" {
		t.Errorf("expected 'http://localhost:9999', got %q", cfg.AdminAddr())
	}
}

func TestToYAML(t *testing.T) {
	cfg := DefaultCmkConfig()
	data, err := cfg.ToYAML()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(data) == 0 {
		t.Error("expected non-empty YAML output")
	}
}
