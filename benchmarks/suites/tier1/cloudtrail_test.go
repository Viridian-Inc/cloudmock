package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudTrailSuite_Metadata(t *testing.T) {
	s := NewCloudTrailSuite()
	assert.Equal(t, "cloudtrail", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCloudTrailSuite_Operations(t *testing.T) {
	s := NewCloudTrailSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateTrail"])
	assert.True(t, names["DescribeTrails"])
	assert.True(t, names["DeleteTrail"])
}
