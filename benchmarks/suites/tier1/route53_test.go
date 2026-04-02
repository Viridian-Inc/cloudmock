package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRoute53Suite_Metadata(t *testing.T) {
	s := NewRoute53Suite()
	assert.Equal(t, "route53", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestRoute53Suite_Operations(t *testing.T) {
	s := NewRoute53Suite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateHostedZone"])
	assert.True(t, names["ListHostedZones"])
	assert.True(t, names["DeleteHostedZone"])
}
