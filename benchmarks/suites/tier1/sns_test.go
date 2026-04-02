package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSNSSuite_Metadata(t *testing.T) {
	s := NewSNSSuite()
	assert.Equal(t, "sns", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestSNSSuite_Operations(t *testing.T) {
	s := NewSNSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 5)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateTopic"])
	assert.True(t, names["Publish"])
	assert.True(t, names["Subscribe"])
	assert.True(t, names["ListTopics"])
	assert.True(t, names["DeleteTopic"])
}
