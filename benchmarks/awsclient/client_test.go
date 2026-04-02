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
