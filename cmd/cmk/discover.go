package main

import (
	"os"
	"path/filepath"
)

// IaCKind represents the type of Infrastructure-as-Code tool detected.
type IaCKind string

const (
	IaCNone      IaCKind = ""
	IaCPulumi    IaCKind = "pulumi"
	IaCTerraform IaCKind = "terraform"
)

// IaCDiscovery holds the result of auto-discovering IaC configuration.
type IaCDiscovery struct {
	Kind IaCKind
	Path string // directory containing the IaC config
}

// DiscoverIaC walks up from startDir looking for IaC configuration files.
// It checks for Pulumi first (higher priority), then Terraform.
func DiscoverIaC(startDir string) IaCDiscovery {
	dir := startDir
	for {
		// Check for Pulumi.yaml or Pulumi.*.yaml
		if hasPulumiConfig(dir) {
			return IaCDiscovery{Kind: IaCPulumi, Path: dir}
		}

		// Check for terraform/ directory with .tf files
		tfDir := filepath.Join(dir, "terraform")
		if hasTerraformConfig(tfDir) {
			return IaCDiscovery{Kind: IaCTerraform, Path: tfDir}
		}

		// Also check infra/pulumi pattern
		infraPulumi := filepath.Join(dir, "infra", "pulumi")
		if hasPulumiConfig(infraPulumi) {
			return IaCDiscovery{Kind: IaCPulumi, Path: infraPulumi}
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return IaCDiscovery{}
}

// hasPulumiConfig checks if a directory contains Pulumi.yaml or Pulumi.*.yaml files.
func hasPulumiConfig(dir string) bool {
	if _, err := os.Stat(filepath.Join(dir, "Pulumi.yaml")); err == nil {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "Pulumi.*.yaml"))
	return len(matches) > 0
}

// hasTerraformConfig checks if a directory contains .tf files.
func hasTerraformConfig(dir string) bool {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return false
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.tf"))
	return len(matches) > 0
}
