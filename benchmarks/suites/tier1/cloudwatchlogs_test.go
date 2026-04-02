package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudWatchLogsSuite_Metadata(t *testing.T) {
	s := NewCloudWatchLogsSuite()
	assert.Equal(t, "cloudwatchlogs", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCloudWatchLogsSuite_Operations(t *testing.T) {
	s := NewCloudWatchLogsSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateLogGroup"])
	assert.True(t, names["DescribeLogGroups"])
	assert.True(t, names["PutLogEvents"])
	assert.True(t, names["DeleteLogGroup"])
}
