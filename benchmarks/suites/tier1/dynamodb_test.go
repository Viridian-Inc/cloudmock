package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDynamoDBSuite_Metadata(t *testing.T) {
	s := NewDynamoDBSuite()
	assert.Equal(t, "dynamodb", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestDynamoDBSuite_Operations(t *testing.T) {
	s := NewDynamoDBSuite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 7)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateTable"])
	assert.True(t, names["PutItem"])
	assert.True(t, names["GetItem"])
	assert.True(t, names["Query"])
	assert.True(t, names["Scan"])
	assert.True(t, names["UpdateItem"])
	assert.True(t, names["DeleteItem"])
}
