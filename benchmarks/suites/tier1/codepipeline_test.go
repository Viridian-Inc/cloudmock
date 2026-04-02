package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodePipelineSuite_Metadata(t *testing.T) {
	s := NewCodePipelineSuite()
	assert.Equal(t, "codepipeline", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCodePipelineSuite_Operations(t *testing.T) {
	s := NewCodePipelineSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreatePipeline"])
	assert.True(t, names["GetPipeline"])
	assert.True(t, names["ListPipelines"])
	assert.True(t, names["DeletePipeline"])
}
