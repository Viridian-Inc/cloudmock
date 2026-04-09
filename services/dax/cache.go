package dax

import (
	"container/list"
	"sync"
	"time"
)

// CacheStats tracks cache hit/miss/eviction counters.
type CacheStats struct {
	ItemHits      int64 `json:"itemHits"`
	ItemMisses    int64 `json:"itemMisses"`
	QueryHits     int64 `json:"queryHits"`
	QueryMisses   int64 `json:"queryMisses"`
	ItemSize      int64 `json:"itemSize"`
	QuerySize     int64 `json:"querySize"`
	Evictions     int64 `json:"evictions"`
	WriteThroughs int64 `json:"writeThroughs"`
	Invalidations int64 `json:"invalidations"`
}

type cacheEntry struct {
	value     any
	expiresAt time.Time
	key       string
}

// Cache is a per-cluster LRU+TTL cache for DynamoDB items and queries.
type Cache struct {
	mu          sync.Mutex
	items       map[string]*list.Element
	itemList    *list.List
	queries     map[string]*cacheEntry
	maxSize     int
	recordTTLMs int64
	queryTTLMs  int64
	stats       CacheStats
}

// NewCache returns a cache with the given max size and TTLs in milliseconds.
func NewCache(maxSize int, recordTTLMs, queryTTLMs int64) *Cache {
	return &Cache{
		items:       make(map[string]*list.Element),
		itemList:    list.New(),
		queries:     make(map[string]*cacheEntry),
		maxSize:     maxSize,
		recordTTLMs: recordTTLMs,
		queryTTLMs:  queryTTLMs,
	}
}

// Stats returns a snapshot of cache counters.
func (c *Cache) Stats() CacheStats {
	c.mu.Lock()
	itemSize := int64(c.itemList.Len())
	querySize := int64(len(c.queries))
	c.mu.Unlock()
	return CacheStats{
		ItemHits:      c.stats.ItemHits,
		ItemMisses:    c.stats.ItemMisses,
		QueryHits:     c.stats.QueryHits,
		QueryMisses:   c.stats.QueryMisses,
		ItemSize:      itemSize,
		QuerySize:     querySize,
		Evictions:     c.stats.Evictions,
		WriteThroughs: c.stats.WriteThroughs,
		Invalidations: c.stats.Invalidations,
	}
}

func itemKey(table, pk, sk string) string {
	return table + "\x00" + pk + "\x00" + sk
}

// SetItem stores an item in the cache with record TTL.
func (c *Cache) SetItem(table, pk, sk string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(c.recordTTLMs) * time.Millisecond),
		key:       key,
	}

	if el, ok := c.items[key]; ok {
		el.Value = entry
		c.itemList.MoveToFront(el)
	} else {
		el := c.itemList.PushFront(entry)
		c.items[key] = el
	}
	c.evictIfNeeded()
}

func (c *Cache) evictIfNeeded() {
	for c.itemList.Len() > c.maxSize {
		back := c.itemList.Back()
		if back == nil {
			break
		}
		entry := back.Value.(*cacheEntry)
		c.itemList.Remove(back)
		delete(c.items, entry.key)
		c.stats.Evictions++
	}
}

// GetItem returns a cached item or nil on miss.
func (c *Cache) GetItem(table, pk, sk string) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	el, ok := c.items[key]
	if !ok {
		c.stats.ItemMisses++
		return nil
	}
	entry := el.Value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.itemList.Remove(el)
		delete(c.items, key)
		c.stats.ItemMisses++
		return nil
	}
	c.itemList.MoveToFront(el)
	c.stats.ItemHits++
	return entry.value
}

// GetQuery returns a cached query result or nil on miss.
func (c *Cache) GetQuery(queryKey string) any {
	c.mu.Lock()
	defer c.mu.Unlock()
	entry, ok := c.queries[queryKey]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.queries, queryKey)
		}
		c.stats.QueryMisses++
		return nil
	}
	c.stats.QueryHits++
	return entry.value
}

// SetQuery stores a query result with query TTL.
func (c *Cache) SetQuery(queryKey string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.queries[queryKey] = &cacheEntry{
		value: value, expiresAt: time.Now().Add(time.Duration(c.queryTTLMs) * time.Millisecond), key: queryKey,
	}
}

// InvalidateItem removes a specific item and all query cache for that table.
func (c *Cache) InvalidateItem(table, pk, sk string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	key := itemKey(table, pk, sk)
	if el, ok := c.items[key]; ok {
		c.itemList.Remove(el)
		delete(c.items, key)
	}
	c.invalidateQueriesForTable(table)
	c.stats.Invalidations++
}

// InvalidateTable removes all items and queries for a table.
func (c *Cache) InvalidateTable(table string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	prefix := table + "\x00"
	for key, el := range c.items {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			c.itemList.Remove(el)
			delete(c.items, key)
		}
	}
	c.invalidateQueriesForTable(table)
	c.stats.Invalidations++
}

// IncrWriteThroughs increments the write-through counter.
func (c *Cache) IncrWriteThroughs() {
	c.mu.Lock()
	c.stats.WriteThroughs++
	c.mu.Unlock()
}

func (c *Cache) invalidateQueriesForTable(table string) {
	prefix := table + "|"
	for key := range c.queries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.queries, key)
		}
	}
}
