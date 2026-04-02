package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerTarget_Config(t *testing.T) {
	tests := []struct {
		name   string
		target string
		image  string
		port   int
	}{
		{"cloudmock", "cloudmock", "ghcr.io/viridian-inc/cloudmock:latest", 4566},
		{"localstack", "localstack", "localstack/localstack:latest", 4566},
		{"localstack-pro", "localstack-pro", "localstack/localstack-pro:latest", 4566},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dt := NewDockerTarget(tt.target, "")
			assert.Equal(t, tt.image, dt.Image())
			assert.Equal(t, tt.port, dt.Port())
			assert.Equal(t, tt.target, dt.Name())
		})
	}
}
