package tier1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestS3Suite_Metadata(t *testing.T) {
	s := NewS3Suite()
	assert.Equal(t, "s3", s.Name())
	assert.Equal(t, 1, s.Tier())
}

func TestS3Suite_Operations(t *testing.T) {
	s := NewS3Suite()
	ops := s.Operations()
	assert.GreaterOrEqual(t, len(ops), 7)

	names := make(map[string]bool)
	for _, op := range ops {
		names[op.Name] = true
	}
	assert.True(t, names["CreateBucket"])
	assert.True(t, names["PutObject"])
	assert.True(t, names["GetObject"])
	assert.True(t, names["ListObjects"])
	assert.True(t, names["CopyObject"])
	assert.True(t, names["DeleteObject"])
	assert.True(t, names["DeleteBucket"])
}
