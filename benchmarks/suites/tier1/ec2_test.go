package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEC2Suite_Metadata(t *testing.T) {
	s := NewEC2Suite()
	assert.Equal(t, "ec2", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestEC2Suite_Operations(t *testing.T) {
	s := NewEC2Suite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["DescribeInstances"])
	assert.True(t, names["DescribeVpcs"])
	assert.True(t, names["DescribeSubnets"])
	assert.True(t, names["DescribeSecurityGroups"])
}
