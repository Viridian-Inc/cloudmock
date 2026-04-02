package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIAMSuite_Metadata(t *testing.T) {
	s := NewIAMSuite()
	assert.Equal(t, "iam", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestIAMSuite_Operations(t *testing.T) {
	s := NewIAMSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 6)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateUser"])
	assert.True(t, names["GetUser"])
	assert.True(t, names["ListUsers"])
	assert.True(t, names["CreateRole"])
	assert.True(t, names["DeleteUser"])
	assert.True(t, names["DeleteRole"])
}
