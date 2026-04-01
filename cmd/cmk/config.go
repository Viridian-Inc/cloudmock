package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

// CmkConfig is the user-facing .cloudmock.yaml configuration.
type CmkConfig struct {
	Gateway   CmkGatewayConfig   `yaml:"gateway"`
	Admin     CmkAdminConfig     `yaml:"admin"`
	Dashboard CmkDashboardConfig `yaml:"dashboard"`
	OTLP      CmkOTLPConfig      `yaml:"otlp"`
	IaC       CmkIaCConfig       `yaml:"iac"`
	Profile   string             `yaml:"profile"`
	Region    string             `yaml:"region"`
	AccountID string             `yaml:"account_id"`
}

// CmkGatewayConfig holds gateway port configuration.
type CmkGatewayConfig struct {
	Port int `yaml:"port"`
}

// CmkAdminConfig holds admin API port configuration.
type CmkAdminConfig struct {
	Port int `yaml:"port"`
}

// CmkDashboardConfig holds dashboard configuration.
type CmkDashboardConfig struct {
	Port    int  `yaml:"port"`
	Enabled bool `yaml:"enabled"`
}

// CmkOTLPConfig holds OpenTelemetry collector configuration.
type CmkOTLPConfig struct {
	Port int `yaml:"port"`
}

// CmkIaCConfig holds Infrastructure-as-Code configuration.
type CmkIaCConfig struct {
	Path string `yaml:"path"`
	Env  string `yaml:"env"`
}

// DefaultCmkConfig returns a CmkConfig populated with sensible defaults.
func DefaultCmkConfig() *CmkConfig {
	return &CmkConfig{
		Gateway:   CmkGatewayConfig{Port: 4566},
		Admin:     CmkAdminConfig{Port: 4599},
		Dashboard: CmkDashboardConfig{Port: 4500, Enabled: true},
		OTLP:      CmkOTLPConfig{Port: 4318},
		Profile:   "standard",
		Region:    "us-east-1",
		AccountID: "000000000000",
	}
}

// LoadCmkConfig loads the .cloudmock.yaml config file from the given path.
// It starts with defaults and overlays the file contents on top.
func LoadCmkConfig(path string) (*CmkConfig, error) {
	cfg := DefaultCmkConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file %s: %w", path, err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file %s: %w", path, err)
	}

	return cfg, nil
}

// FindConfigFile walks up from startDir looking for .cloudmock.yaml.
// Returns the path if found, or empty string if not found.
func FindConfigFile(startDir string) string {
	dir := startDir
	for {
		candidate := filepath.Join(dir, ".cloudmock.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// ApplyEnvOverrides applies environment variable overrides to the config.
func (c *CmkConfig) ApplyEnvOverrides() {
	if v := os.Getenv("CLOUDMOCK_GATEWAY_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Gateway.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_ADMIN_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Admin.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_DASHBOARD_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			c.Dashboard.Port = p
		}
	}
	if v := os.Getenv("CLOUDMOCK_REGION"); v != "" {
		c.Region = v
	}
	if v := os.Getenv("CLOUDMOCK_PROFILE"); v != "" {
		c.Profile = v
	}
}

// AdminAddr returns the full admin API base URL.
func (c *CmkConfig) AdminAddr() string {
	return fmt.Sprintf("http://localhost:%d", c.Admin.Port)
}

// ToYAML serializes the config to YAML bytes.
func (c *CmkConfig) ToYAML() ([]byte, error) {
	return yaml.Marshal(c)
}
