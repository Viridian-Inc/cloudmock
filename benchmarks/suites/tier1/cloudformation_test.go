package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudFormationSuite_Metadata(t *testing.T) {
	s := NewCloudFormationSuite()
	assert.Equal(t, "cloudformation", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCloudFormationSuite_Operations(t *testing.T) {
	s := NewCloudFormationSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateStack"])
	assert.True(t, names["DescribeStacks"])
	assert.True(t, names["ListStacks"])
	assert.True(t, names["DeleteStack"])
}
