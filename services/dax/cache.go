package dax

import (
	"sync"
	"sync/atomic"
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
	items       map[string]*cacheEntry
	queries     map[string]*cacheEntry
	maxSize     int
	recordTTLMs int64
	queryTTLMs  int64
	stats       CacheStats
}

// NewCache returns a cache with the given max size and TTLs in milliseconds.
func NewCache(maxSize int, recordTTLMs, queryTTLMs int64) *Cache {
	return &Cache{
		items:       make(map[string]*cacheEntry),
		queries:     make(map[string]*cacheEntry),
		maxSize:     maxSize,
		recordTTLMs: recordTTLMs,
		queryTTLMs:  queryTTLMs,
	}
}

// Stats returns a snapshot of cache counters.
func (c *Cache) Stats() CacheStats {
	return CacheStats{
		ItemHits:      atomic.LoadInt64(&c.stats.ItemHits),
		ItemMisses:    atomic.LoadInt64(&c.stats.ItemMisses),
		QueryHits:     atomic.LoadInt64(&c.stats.QueryHits),
		QueryMisses:   atomic.LoadInt64(&c.stats.QueryMisses),
		ItemSize:      int64(len(c.items)),
		QuerySize:     int64(len(c.queries)),
		Evictions:     atomic.LoadInt64(&c.stats.Evictions),
		WriteThroughs: atomic.LoadInt64(&c.stats.WriteThroughs),
		Invalidations: atomic.LoadInt64(&c.stats.Invalidations),
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
	c.items[key] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(c.recordTTLMs) * time.Millisecond),
		key:       key,
	}
	c.evictIfNeeded()
}

func (c *Cache) evictIfNeeded() {
	for len(c.items) > c.maxSize {
		for k := range c.items {
			delete(c.items, k)
			atomic.AddInt64(&c.stats.Evictions, 1)
			break
		}
	}
}

// GetItem returns a cached item or nil on miss.
func (c *Cache) GetItem(table, pk, sk string) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.items, key)
		}
		atomic.AddInt64(&c.stats.ItemMisses, 1)
		return nil
	}
	atomic.AddInt64(&c.stats.ItemHits, 1)
	return entry.value
}
