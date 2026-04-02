package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestECSSuite_Metadata(t *testing.T) {
	s := NewECSSuite()
	assert.Equal(t, "ecs", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestECSSuite_Operations(t *testing.T) {
	s := NewECSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateCluster"])
	assert.True(t, names["DescribeClusters"])
	assert.True(t, names["ListClusters"])
	assert.True(t, names["DeleteCluster"])
}
