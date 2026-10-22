package awsclient

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfig(t *testing.T) {
	cfg, err := NewConfig("http://localhost:4566")
	require.NoError(t, err)
	assert.Equal(t, "us-east-1", cfg.Region)

	creds, err := cfg.Credentials.Retrieve(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "test", creds.AccessKeyID)
	assert.Equal(t, "test", creds.SecretAccessKey)
}

func TestNewConfigPreservesNonlegacyDefaults(t *testing.T) {
	for _, mode := range []string{"legacy", "standard", "in-region"} {
		t.Run(mode, func(t *testing.T) {
			t.Setenv("AWS_DEFAULTS_MODE", mode)
			cfg, err := NewConfig("http://localhost:4566")
			require.NoError(t, err)
			_, wrapped := cfg.HTTPClient.(*coalescingClient)
			assert.Equal(t, mode == "legacy", wrapped)
			assert.Equal(t, mode, string(cfg.DefaultsMode))
		})
	}
}
