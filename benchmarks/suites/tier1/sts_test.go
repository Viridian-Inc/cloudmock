package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSTSSuite_Metadata(t *testing.T) {
	s := NewSTSSuite()
	assert.Equal(t, "sts", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestSTSSuite_Operations(t *testing.T) {
	s := NewSTSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 2)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["GetCallerIdentity"])
	assert.True(t, names["AssumeRole"])
}
