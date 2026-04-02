package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKMSSuite_Metadata(t *testing.T) {
	s := NewKMSSuite()
	assert.Equal(t, "kms", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestKMSSuite_Operations(t *testing.T) {
	s := NewKMSSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateKey"])
	assert.True(t, names["Encrypt"])
	assert.True(t, names["Decrypt"])
	assert.True(t, names["ListKeys"])
}
