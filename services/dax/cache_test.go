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

func TestCache_LRUEviction(t *testing.T) {
	c := NewCache(3, 300000, 300000) // max 3 items
	c.SetItem("t", "k1", "", "v1")
	c.SetItem("t", "k2", "", "v2")
	c.SetItem("t", "k3", "", "v3")

	// Access k1 to make it recently used
	c.GetItem("t", "k1", "")

	// Insert k4 — should evict k2 (least recently used)
	c.SetItem("t", "k4", "", "v4")

	assert.NotNil(t, c.GetItem("t", "k1", ""), "k1 should survive (recently accessed)")
	assert.Nil(t, c.GetItem("t", "k2", ""), "k2 should be evicted (LRU)")
	assert.NotNil(t, c.GetItem("t", "k3", ""))
	assert.NotNil(t, c.GetItem("t", "k4", ""))
	assert.True(t, c.Stats().Evictions >= 1)
}

func TestCache_QueryCacheHit(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	results := []any{map[string]any{"pk": "1"}, map[string]any{"pk": "2"}}
	c.SetQuery("users|idx|pk=1", results)
	val := c.GetQuery("users|idx|pk=1")
	assert.NotNil(t, val)
	assert.Len(t, val.([]any), 2)
	assert.Equal(t, int64(1), c.Stats().QueryHits)
}

func TestCache_InvalidateTable(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	c.SetItem("users", "pk1", "", "val1")
	c.SetItem("users", "pk2", "", "val2")
	c.SetItem("orders", "pk1", "", "val3")
	c.SetQuery("users|idx|pk=1", []any{"result"})
	c.InvalidateTable("users")
	assert.Nil(t, c.GetItem("users", "pk1", ""))
	assert.Nil(t, c.GetItem("users", "pk2", ""))
	assert.NotNil(t, c.GetItem("orders", "pk1", ""))
	assert.Nil(t, c.GetQuery("users|idx|pk=1"))
}

func TestCache_InvalidateItem(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	c.SetItem("users", "pk1", "sk1", "val1")
	c.SetItem("users", "pk2", "", "val2")
	c.SetQuery("users|idx|pk=1", []any{"result"})
	c.InvalidateItem("users", "pk1", "sk1")
	assert.Nil(t, c.GetItem("users", "pk1", "sk1"))
	assert.NotNil(t, c.GetItem("users", "pk2", ""))
	assert.Nil(t, c.GetQuery("users|idx|pk=1"))
}
