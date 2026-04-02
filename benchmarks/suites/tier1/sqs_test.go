package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSQSSuite_Metadata(t *testing.T) {
	s := NewSQSSuite()
	assert.Equal(t, "sqs", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestSQSSuite_Operations(t *testing.T) {
	s := NewSQSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 5)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateQueue"])
	assert.True(t, names["SendMessage"])
	assert.True(t, names["ReceiveMessage"])
	assert.True(t, names["DeleteMessage"])
	assert.True(t, names["DeleteQueue"])
}
