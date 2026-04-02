package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNativeTarget_Config(t *testing.T) {
	nt := NewNativeTarget(4566)
	assert.Equal(t, "cloudmock-native", nt.Name())
	assert.Equal(t, "http://localhost:4566", nt.Endpoint())
}
