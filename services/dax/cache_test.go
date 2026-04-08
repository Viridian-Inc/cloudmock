package dax

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_GetMiss(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	val := c.GetItem("users", "pk1", "sk1")
	assert.Nil(t, val)
	assert.Equal(t, int64(0), c.Stats().ItemHits)
	assert.Equal(t, int64(1), c.Stats().ItemMisses)
}
