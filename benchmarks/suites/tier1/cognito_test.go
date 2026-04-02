package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCognitoSuite_Metadata(t *testing.T) {
	s := NewCognitoSuite()
	assert.Equal(t, "cognito", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestCognitoSuite_Operations(t *testing.T) {
	s := NewCognitoSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateUserPool"])
	assert.True(t, names["DescribeUserPool"])
	assert.True(t, names["ListUserPools"])
	assert.True(t, names["DeleteUserPool"])
}
