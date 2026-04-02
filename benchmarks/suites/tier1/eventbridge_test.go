package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventBridgeSuite_Metadata(t *testing.T) {
	s := NewEventBridgeSuite()
	assert.Equal(t, "eventbridge", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestEventBridgeSuite_Operations(t *testing.T) {
	s := NewEventBridgeSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 4)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateEventBus"])
	assert.True(t, names["PutRule"])
	assert.True(t, names["PutEvents"])
	assert.True(t, names["DeleteEventBus"])
}
