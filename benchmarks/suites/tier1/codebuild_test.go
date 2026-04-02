package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCodeBuildSuite_Metadata(t *testing.T) {
	s := NewCodeBuildSuite()
	assert.Equal(t, "codebuild", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCodeBuildSuite_Operations(t *testing.T) {
	s := NewCodeBuildSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateProject"])
	assert.True(t, names["ListProjects"])
	assert.True(t, names["DeleteProject"])
}
