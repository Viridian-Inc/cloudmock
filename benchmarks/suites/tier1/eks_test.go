package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEKSSuite_Metadata(t *testing.T) {
	s := NewEKSSuite()
	assert.Equal(t, "eks", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestEKSSuite_Operations(t *testing.T) {
	s := NewEKSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateCluster"])
	assert.True(t, names["DescribeCluster"])
	assert.True(t, names["ListClusters"])
	assert.True(t, names["DeleteCluster"])
}
