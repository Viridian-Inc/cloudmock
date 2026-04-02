package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCloudWatchSuite_Metadata(t *testing.T) {
	s := NewCloudWatchSuite()
	assert.Equal(t, "cloudwatch", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCloudWatchSuite_Operations(t *testing.T) {
	s := NewCloudWatchSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["PutMetricData"])
	assert.True(t, names["GetMetricData"])
	assert.True(t, names["ListMetrics"])
}
