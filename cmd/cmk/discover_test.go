package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverIaC_Pulumi(t *testing.T) {
	dir := t.TempDir()

	// Create Pulumi.yaml
	if err := os.WriteFile(filepath.Join(dir, "Pulumi.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(dir)
	if disc.Kind != IaCPulumi {
		t.Errorf("expected IaCPulumi, got %q", disc.Kind)
	}
	if disc.Path != dir {
		t.Errorf("expected path %q, got %q", dir, disc.Path)
	}
}

func TestDiscoverIaC_PulumiStacked(t *testing.T) {
	dir := t.TempDir()

	// Create Pulumi.dev.yaml (stack file)
	if err := os.WriteFile(filepath.Join(dir, "Pulumi.dev.yaml"), []byte("config: {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(dir)
	if disc.Kind != IaCPulumi {
		t.Errorf("expected IaCPulumi, got %q", disc.Kind)
	}
}

func TestDiscoverIaC_InfraPulumi(t *testing.T) {
	root := t.TempDir()
	infraDir := filepath.Join(root, "infra", "pulumi")
	if err := os.MkdirAll(infraDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(infraDir, "Pulumi.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(root)
	if disc.Kind != IaCPulumi {
		t.Errorf("expected IaCPulumi, got %q", disc.Kind)
	}
	if disc.Path != infraDir {
		t.Errorf("expected path %q, got %q", infraDir, disc.Path)
	}
}

func TestDiscoverIaC_Terraform(t *testing.T) {
	root := t.TempDir()
	tfDir := filepath.Join(root, "terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("resource {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(root)
	if disc.Kind != IaCTerraform {
		t.Errorf("expected IaCTerraform, got %q", disc.Kind)
	}
	if disc.Path != tfDir {
		t.Errorf("expected path %q, got %q", tfDir, disc.Path)
	}
}

func TestDiscoverIaC_PulumiPreferredOverTerraform(t *testing.T) {
	root := t.TempDir()

	// Create both Pulumi and Terraform
	if err := os.WriteFile(filepath.Join(root, "Pulumi.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	tfDir := filepath.Join(root, "terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tfDir, "main.tf"), []byte("resource {}\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(root)
	if disc.Kind != IaCPulumi {
		t.Errorf("expected Pulumi to be preferred, got %q", disc.Kind)
	}
}

func TestDiscoverIaC_WalksUp(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "src", "app")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Pulumi.yaml"), []byte("name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	disc := DiscoverIaC(nested)
	if disc.Kind != IaCPulumi {
		t.Errorf("expected IaCPulumi, got %q", disc.Kind)
	}
	if disc.Path != root {
		t.Errorf("expected path %q, got %q", root, disc.Path)
	}
}

func TestDiscoverIaC_NoneFound(t *testing.T) {
	dir := t.TempDir()
	disc := DiscoverIaC(dir)
	if disc.Kind != IaCNone {
		t.Errorf("expected IaCNone, got %q", disc.Kind)
	}
	if disc.Path != "" {
		t.Errorf("expected empty path, got %q", disc.Path)
	}
}

func TestDiscoverIaC_EmptyTerraformDir(t *testing.T) {
	root := t.TempDir()
	tfDir := filepath.Join(root, "terraform")
	if err := os.MkdirAll(tfDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Empty terraform directory (no .tf files) should not match
	disc := DiscoverIaC(root)
	if disc.Kind != IaCNone {
		t.Errorf("expected IaCNone for empty terraform dir, got %q", disc.Kind)
	}
}
