package tier2

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateSuites_Count(t *testing.T) {
	suites := GenerateAll()
	assert.GreaterOrEqual(t, len(suites), 73)
}

func TestGenerateSuites_Tier(t *testing.T) {
	suites := GenerateAll()
	for _, s := range suites {
		assert.Equal(t, 2, s.Tier(), "suite %s should be tier 2", s.Name())
	}
}

func TestGenerateSuites_HasOperations(t *testing.T) {
	suites := GenerateAll()
	require.NotEmpty(t, suites)
	for _, s := range suites {
		ops := s.Operations()
		assert.GreaterOrEqual(t, len(ops), 1, "suite %s should have >=1 operations", s.Name())
	}
}

func TestGenerateSuites_OperationNames(t *testing.T) {
	suites := GenerateAll()
	var found bool
	for _, s := range suites {
		if s.Name() == "acm" {
			found = true
			ops := s.Operations()
			names := make(map[string]bool)
			for _, op := range ops {
				names[op.Name] = true
			}
			assert.True(t, names["ListCertificates"], "acm should have ListCertificates")
			break
		}
	}
	assert.True(t, found, "should find acm in generated suites")
}
