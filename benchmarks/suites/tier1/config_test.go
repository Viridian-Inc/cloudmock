package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigSuite_Metadata(t *testing.T) {
	s := NewConfigSuite()
	assert.Equal(t, "config", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestConfigSuite_Operations(t *testing.T) {
	s := NewConfigSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["PutConfigRule"])
	assert.True(t, names["DescribeConfigRules"])
	assert.True(t, names["DeleteConfigRule"])
}
