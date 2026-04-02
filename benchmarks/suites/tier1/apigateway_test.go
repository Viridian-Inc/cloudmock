package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAPIGatewaySuite_Metadata(t *testing.T) {
	s := NewAPIGatewaySuite()
	assert.Equal(t, "apigateway", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestAPIGatewaySuite_Operations(t *testing.T) {
	s := NewAPIGatewaySuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateRestApi"])
	assert.True(t, names["GetRestApi"])
	assert.True(t, names["GetRestApis"])
	assert.True(t, names["DeleteRestApi"])
}
