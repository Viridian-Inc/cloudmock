package stress

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDynamoDBStressSuite_Metadata(t *testing.T) {
	s := NewDynamoDBStressSuite()
	assert.Equal(t, "dynamodb-stress", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestDynamoDBStressSuite_Operations(t *testing.T) {
	s := NewDynamoDBStressSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 3)
}
