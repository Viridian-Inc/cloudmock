package dax

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCache_GetMiss(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	val := c.GetItem("users", "pk1", "sk1")
	assert.Nil(t, val)
	assert.Equal(t, int64(0), c.Stats().ItemHits)
	assert.Equal(t, int64(1), c.Stats().ItemMisses)
}

func TestCache_SetThenGetHit(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	item := map[string]any{"pk": "user1", "name": "Alice"}
	c.SetItem("users", "user1", "", item)

	val := c.GetItem("users", "user1", "")
	assert.NotNil(t, val)
	assert.Equal(t, "Alice", val.(map[string]any)["name"])
	assert.Equal(t, int64(1), c.Stats().ItemHits)
	assert.Equal(t, int64(0), c.Stats().ItemMisses)
}

func TestCache_ItemExpiry(t *testing.T) {
	c := NewCache(1000, 50, 50) // 50ms TTL
	c.SetItem("users", "pk1", "", map[string]any{"name": "Alice"})

	val := c.GetItem("users", "pk1", "")
	assert.NotNil(t, val)

	time.Sleep(60 * time.Millisecond)

	val = c.GetItem("users", "pk1", "")
	assert.Nil(t, val, "expected nil after TTL expiry")
	assert.Equal(t, int64(1), c.Stats().ItemHits)
	assert.Equal(t, int64(1), c.Stats().ItemMisses)
}
