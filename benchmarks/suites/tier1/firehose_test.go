package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFirehoseSuite_Metadata(t *testing.T) {
	s := NewFirehoseSuite()
	assert.Equal(t, "firehose", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestFirehoseSuite_Operations(t *testing.T) {
	s := NewFirehoseSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateDeliveryStream"])
	assert.True(t, names["DescribeDeliveryStream"])
	assert.True(t, names["DeleteDeliveryStream"])
}
