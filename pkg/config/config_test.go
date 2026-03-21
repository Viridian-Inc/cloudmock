package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/neureaux/cloudmock/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := config.Default()
	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "000000000000", cfg.AccountID)
	assert.Equal(t, "test", cfg.IAM.RootAccessKey)
	assert.Equal(t, "test", cfg.IAM.RootSecretKey)
	assert.Equal(t, "enforce", cfg.IAM.Mode)
	assert.False(t, cfg.Persistence.Enabled)
	assert.True(t, cfg.Dashboard.Enabled)
	assert.Equal(t, 4566, cfg.Gateway.Port)
	assert.Equal(t, 4500, cfg.Dashboard.Port)
	assert.Equal(t, 4599, cfg.Admin.Port)
	assert.Equal(t, "minimal", cfg.Profile)
}

func TestLoadFromFile(t *testing.T) {
	yaml := `
region: us-west-2
account_id: "123456789012"
profile: standard
iam:
  mode: none
  root_access_key: mykey
  root_secret_key: mysecret
persistence:
  enabled: true
  path: /tmp/cloudmock-data
`
	tmpFile := filepath.Join(t.TempDir(), "cloudmock.yml")
	require.NoError(t, os.WriteFile(tmpFile, []byte(yaml), 0644))

	cfg, err := config.LoadFromFile(tmpFile)
	require.NoError(t, err)
	assert.Equal(t, "us-west-2", cfg.Region)
	assert.Equal(t, "123456789012", cfg.AccountID)
	assert.Equal(t, "standard", cfg.Profile)
	assert.Equal(t, "none", cfg.IAM.Mode)
	assert.Equal(t, "mykey", cfg.IAM.RootAccessKey)
	assert.True(t, cfg.Persistence.Enabled)
	assert.Equal(t, "/tmp/cloudmock-data", cfg.Persistence.Path)
}

func TestEnvOverrides(t *testing.T) {
	t.Setenv("CLOUDMOCK_REGION", "eu-west-1")
	t.Setenv("CLOUDMOCK_IAM_MODE", "authenticate")
	t.Setenv("CLOUDMOCK_PERSIST", "true")
	t.Setenv("CLOUDMOCK_LOG_LEVEL", "debug")

	cfg := config.Default()
	cfg.ApplyEnv()

	assert.Equal(t, "eu-west-1", cfg.Region)
	assert.Equal(t, "authenticate", cfg.IAM.Mode)
	assert.True(t, cfg.Persistence.Enabled)
	assert.Equal(t, "debug", cfg.Logging.Level)
}

func TestProfileServices(t *testing.T) {
	cfg := config.Default()
	cfg.Profile = "minimal"
	services := cfg.EnabledServices()
	assert.Contains(t, services, "iam")
	assert.Contains(t, services, "s3")
	assert.Contains(t, services, "sqs")
	assert.Contains(t, services, "lambda")
	assert.NotContains(t, services, "rds")

	cfg.Profile = "standard"
	services = cfg.EnabledServices()
	assert.Contains(t, services, "rds")
	assert.Contains(t, services, "cloudformation")
}
