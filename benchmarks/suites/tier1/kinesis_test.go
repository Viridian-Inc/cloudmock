package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKinesisSuite_Metadata(t *testing.T) {
	s := NewKinesisSuite()
	assert.Equal(t, "kinesis", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestKinesisSuite_Operations(t *testing.T) {
	s := NewKinesisSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateStream"])
	assert.True(t, names["DescribeStream"])
	assert.True(t, names["PutRecord"])
	assert.True(t, names["DeleteStream"])
}
