package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLambdaSuite_Metadata(t *testing.T) {
	s := NewLambdaSuite()
	assert.Equal(t, "lambda", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestLambdaSuite_Operations(t *testing.T) {
	s := NewLambdaSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateFunction"])
	assert.True(t, names["GetFunction"])
	assert.True(t, names["ListFunctions"])
	assert.True(t, names["DeleteFunction"])
}
