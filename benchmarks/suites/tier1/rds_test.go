package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRDSSuite_Metadata(t *testing.T) {
	s := NewRDSSuite()
	assert.Equal(t, "rds", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestRDSSuite_Operations(t *testing.T) {
	s := NewRDSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateDBInstance"])
	assert.True(t, names["DescribeDBInstances"])
	assert.True(t, names["DeleteDBInstance"])
}
