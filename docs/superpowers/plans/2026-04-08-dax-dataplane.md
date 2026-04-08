# DAX Data-Plane Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a DAX data-plane HTTP service to cloudmock that acts as a caching proxy in front of the existing DynamoDB service, supporting read-through/write-through with configurable TTL and invalidation.

**Architecture:** A new HTTP server on port `:8111` accepts standard DynamoDB JSON requests routed through a per-cluster LRU+TTL cache. Cache misses forward to the existing `DynamoDBService.HandleRequest()` in-process. Write operations pass through to DynamoDB and then invalidate or update the cache based on cluster config. A `/stats/{cluster}` endpoint exposes cache metrics.

**Tech Stack:** Go 1.26, `net/http`, `sync.Map`, `container/list` (LRU), `github.com/stretchr/testify`, existing `pkg/service` framework.

---

## File Map

| File | Responsibility |
|---|---|
| `services/dax/cache.go` | LRU+TTL cache with item and query namespaces |
| `services/dax/cache_test.go` | Cache unit tests: get/set, TTL expiry, LRU eviction, invalidation |
| `services/dax/dataplane.go` | HTTP handlers proxying DynamoDB operations through cache |
| `services/dax/dataplane_test.go` | Integration tests: full read-through/write-through flows |
| `services/dax/store.go` | Modified: add `GetClusterCache()` and `write-strategy` parameter |
| `cmd/gateway/main.go` | Modified: start DAX data-plane HTTP server on `:8111` |
| `website/src/content/docs/docs/services/dax.md` | Modified: document data-plane usage |

---

### Task 1: Cache Store — Core Get/Set with TTL

**Files:**
- Create: `services/dax/cache.go`
- Create: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing test for cache miss returning nil**

```go
// cache_test.go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_GetMiss -v`
Expected: FAIL — `NewCache` undefined

- [ ] **Step 3: Write minimal Cache struct with GetItem**

```go
// cache.go
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
	mu            sync.Mutex
	items         map[string]*cacheEntry
	queries       map[string]*cacheEntry
	maxSize       int
	recordTTLMs   int64
	queryTTLMs    int64
	stats         CacheStats
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

// GetItem returns a cached item or nil on miss.
func (c *Cache) GetItem(table, pk, sk string) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.items, key) // expired
		}
		atomic.AddInt64(&c.stats.ItemMisses, 1)
		return nil
	}
	atomic.AddInt64(&c.stats.ItemHits, 1)
	return entry.value
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_GetMiss -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/cache.go services/dax/cache_test.go && git commit -m "feat(dax): add cache store with GetItem miss path"
```

---

### Task 2: Cache Store — SetItem and Hit Path

**Files:**
- Modify: `services/dax/cache.go`
- Modify: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing test for set then get (hit)**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_SetThenGetHit -v`
Expected: FAIL — `SetItem` undefined

- [ ] **Step 3: Implement SetItem**

Add to `cache.go`:

```go
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
		// Evict oldest entry (simple: pick any — LRU ordering added in Task 3)
		for k := range c.items {
			delete(c.items, k)
			atomic.AddInt64(&c.stats.Evictions, 1)
			break
		}
	}
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_SetThenGetHit -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/cache.go services/dax/cache_test.go && git commit -m "feat(dax): add SetItem with TTL and hit path"
```

---

### Task 3: Cache Store — TTL Expiry

**Files:**
- Modify: `services/dax/cache.go`
- Modify: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing test for TTL expiry**

```go
func TestCache_ItemExpiry(t *testing.T) {
	c := NewCache(1000, 50, 50) // 50ms TTL
	c.SetItem("users", "pk1", "", map[string]any{"name": "Alice"})

	// Immediate read — should hit
	val := c.GetItem("users", "pk1", "")
	assert.NotNil(t, val)

	// Wait for expiry
	time.Sleep(60 * time.Millisecond)

	val = c.GetItem("users", "pk1", "")
	assert.Nil(t, val, "expected nil after TTL expiry")
	assert.Equal(t, int64(1), c.Stats().ItemHits)
	assert.Equal(t, int64(1), c.Stats().ItemMisses)
}
```

- [ ] **Step 2: Run test to verify it passes (TTL already implemented in GetItem)**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_ItemExpiry -v`
Expected: PASS (TTL check is already in `GetItem` from Task 1)

- [ ] **Step 3: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/cache_test.go && git commit -m "test(dax): add TTL expiry test"
```

---

### Task 4: Cache Store — LRU Eviction

**Files:**
- Modify: `services/dax/cache.go`
- Modify: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing test for LRU eviction**

```go
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache_LRUEviction -v`
Expected: FAIL — current eviction is random, not LRU

- [ ] **Step 3: Implement LRU ordering with container/list**

Replace the items map and eviction logic in `cache.go`:

```go
import (
	"container/list"
	"sync"
	"sync/atomic"
	"time"
)

type Cache struct {
	mu            sync.Mutex
	items         map[string]*list.Element
	itemList      *list.List // front = most recent, back = LRU
	queries       map[string]*cacheEntry
	maxSize       int
	recordTTLMs   int64
	queryTTLMs    int64
	stats         CacheStats
}

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

func (c *Cache) GetItem(table, pk, sk string) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	el, ok := c.items[key]
	if !ok {
		atomic.AddInt64(&c.stats.ItemMisses, 1)
		return nil
	}
	entry := el.Value.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		c.itemList.Remove(el)
		delete(c.items, key)
		atomic.AddInt64(&c.stats.ItemMisses, 1)
		return nil
	}
	c.itemList.MoveToFront(el)
	atomic.AddInt64(&c.stats.ItemHits, 1)
	return entry.value
}

func (c *Cache) SetItem(table, pk, sk string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	if el, ok := c.items[key]; ok {
		c.itemList.MoveToFront(el)
		entry := el.Value.(*cacheEntry)
		entry.value = value
		entry.expiresAt = time.Now().Add(time.Duration(c.recordTTLMs) * time.Millisecond)
		return
	}
	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(c.recordTTLMs) * time.Millisecond),
		key:       key,
	}
	el := c.itemList.PushFront(entry)
	c.items[key] = el
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
		atomic.AddInt64(&c.stats.Evictions, 1)
	}
}
```

Update `Stats()` to use `c.itemList.Len()`:

```go
func (c *Cache) Stats() CacheStats {
	c.mu.Lock()
	defer c.mu.Unlock()
	return CacheStats{
		ItemHits:      atomic.LoadInt64(&c.stats.ItemHits),
		ItemMisses:    atomic.LoadInt64(&c.stats.ItemMisses),
		QueryHits:     atomic.LoadInt64(&c.stats.QueryHits),
		QueryMisses:   atomic.LoadInt64(&c.stats.QueryMisses),
		ItemSize:      int64(c.itemList.Len()),
		QuerySize:     int64(len(c.queries)),
		Evictions:     atomic.LoadInt64(&c.stats.Evictions),
		WriteThroughs: atomic.LoadInt64(&c.stats.WriteThroughs),
		Invalidations: atomic.LoadInt64(&c.stats.Invalidations),
	}
}
```

- [ ] **Step 4: Run all cache tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/cache.go services/dax/cache_test.go && git commit -m "feat(dax): implement LRU eviction ordering"
```

---

### Task 5: Cache Store — Query Cache and Table Invalidation

**Files:**
- Modify: `services/dax/cache.go`
- Modify: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing tests for query cache and write invalidation**

```go
func TestCache_QueryCacheHit(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	results := []any{map[string]any{"pk": "1"}, map[string]any{"pk": "2"}}
	queryKey := "users|idx|pk=1"
	c.SetQuery(queryKey, results)

	val := c.GetQuery(queryKey)
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
	assert.NotNil(t, c.GetItem("orders", "pk1", ""), "orders table should be unaffected")
	assert.Nil(t, c.GetQuery("users|idx|pk=1"), "query cache for users should be invalidated")
}

func TestCache_InvalidateItem(t *testing.T) {
	c := NewCache(1000, 300000, 300000)
	c.SetItem("users", "pk1", "sk1", "val1")
	c.SetItem("users", "pk2", "", "val2")
	c.SetQuery("users|idx|pk=1", []any{"result"})

	c.InvalidateItem("users", "pk1", "sk1")

	assert.Nil(t, c.GetItem("users", "pk1", "sk1"))
	assert.NotNil(t, c.GetItem("users", "pk2", ""))
	// Query cache for same table invalidated (conservative, matches real DAX)
	assert.Nil(t, c.GetQuery("users|idx|pk=1"))
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run "TestCache_Query|TestCache_Invalidate" -v`
Expected: FAIL — `SetQuery`, `GetQuery`, `InvalidateTable`, `InvalidateItem` undefined

- [ ] **Step 3: Implement query cache and invalidation methods**

Add to `cache.go`:

```go
// GetQuery returns a cached query result or nil on miss.
func (c *Cache) GetQuery(queryKey string) any {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.queries[queryKey]
	if !ok || time.Now().After(entry.expiresAt) {
		if ok {
			delete(c.queries, queryKey)
		}
		atomic.AddInt64(&c.stats.QueryMisses, 1)
		return nil
	}
	atomic.AddInt64(&c.stats.QueryHits, 1)
	return entry.value
}

// SetQuery stores a query result in the cache with query TTL.
func (c *Cache) SetQuery(queryKey string, value any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.queries[queryKey] = &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(time.Duration(c.queryTTLMs) * time.Millisecond),
		key:       queryKey,
	}
}

// InvalidateItem removes a specific item and all query cache entries for that table.
func (c *Cache) InvalidateItem(table, pk, sk string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	key := itemKey(table, pk, sk)
	if el, ok := c.items[key]; ok {
		c.itemList.Remove(el)
		delete(c.items, key)
	}
	c.invalidateQueriesForTable(table)
	atomic.AddInt64(&c.stats.Invalidations, 1)
}

// InvalidateTable removes all items and query cache entries for a table.
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
	atomic.AddInt64(&c.stats.Invalidations, 1)
}

func (c *Cache) invalidateQueriesForTable(table string) {
	prefix := table + "|"
	for key := range c.queries {
		if len(key) >= len(prefix) && key[:len(prefix)] == prefix {
			delete(c.queries, key)
		}
	}
}
```

- [ ] **Step 4: Run all cache tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestCache -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/cache.go services/dax/cache_test.go && git commit -m "feat(dax): add query cache and table/item invalidation"
```

---

### Task 6: Wire Cache into DAX Store — Per-Cluster Cache Instances

**Files:**
- Modify: `services/dax/store.go`
- Modify: `services/dax/cache_test.go`

- [ ] **Step 1: Write failing test for per-cluster cache retrieval**

```go
func TestStore_GetClusterCache(t *testing.T) {
	s := NewStore("123456789012", "us-east-1")
	_, err := s.CreateCluster("test-cluster", "", "dax.r4.large", 1, "", "", "arn:aws:iam::123456789012:role/r", nil, nil, false, nil)
	require.NoError(t, err)

	cache := s.GetClusterCache("test-cluster")
	assert.NotNil(t, cache)

	// Default TTLs from DescribeDefaultParameters
	cache.SetItem("t", "pk", "", "val")
	assert.NotNil(t, cache.GetItem("t", "pk", ""))

	// Missing cluster returns a default cache
	defaultCache := s.GetClusterCache("nonexistent")
	assert.NotNil(t, defaultCache)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestStore_GetClusterCache -v`
Expected: FAIL — `GetClusterCache` undefined

- [ ] **Step 3: Add GetClusterCache and write-strategy to store.go**

Add to `store.go`:

```go
// Add to Store struct:
//   caches map[string]*Cache

// Update NewStore to init caches map:
//   caches: make(map[string]*Cache),

// GetClusterCache returns the cache for a cluster, creating it on first access.
// TTLs are sourced from the cluster's parameter group settings.
func (s *Store) GetClusterCache(clusterName string) *Cache {
	s.mu.Lock()
	defer s.mu.Unlock()

	if c, ok := s.caches[clusterName]; ok {
		return c
	}

	recordTTL := int64(300000) // default 5 min
	queryTTL := int64(300000)
	maxSize := 10000

	if cluster, ok := s.clusters[clusterName]; ok && cluster.ParameterGroupName != "" {
		if pg, ok := s.parameterGroups[cluster.ParameterGroupName]; ok {
			if v, ok := pg.Parameters["record-ttl-millis"]; ok {
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
					recordTTL = parsed
				}
			}
			if v, ok := pg.Parameters["query-ttl-millis"]; ok {
				if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
					queryTTL = parsed
				}
			}
		}
	}

	cache := NewCache(maxSize, recordTTL, queryTTL)
	s.caches[clusterName] = cache
	return cache
}

// GetWriteStrategy returns the write strategy for a cluster ("invalidate" or "update-cache").
func (s *Store) GetWriteStrategy(clusterName string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if cluster, ok := s.clusters[clusterName]; ok && cluster.ParameterGroupName != "" {
		if pg, ok := s.parameterGroups[cluster.ParameterGroupName]; ok {
			if v, ok := pg.Parameters["write-strategy"]; ok {
				return v
			}
		}
	}
	return "invalidate"
}
```

Add `"strconv"` to `store.go` imports. Add `caches: make(map[string]*Cache)` to `NewStore`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestStore_GetClusterCache -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/store.go services/dax/cache_test.go && git commit -m "feat(dax): add per-cluster cache instances with parameter group TTLs"
```

---

### Task 7: Data-Plane — GetItem Read-Through

**Files:**
- Create: `services/dax/dataplane.go`
- Create: `services/dax/dataplane_test.go`

- [ ] **Step 1: Write failing integration test for GetItem read-through**

```go
// dataplane_test.go
package dax_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	"github.com/Viridian-Inc/cloudmock/services/dax"
	dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupDataPlane(t *testing.T) (*dax.DataPlane, *dynamodbsvc.DynamoDBService) {
	t.Helper()
	ddbSvc := dynamodbsvc.New("123456789012", "us-east-1")
	daxSvc := dax.New("123456789012", "us-east-1")

	// Create a test table
	createTable(t, ddbSvc, "TestTable", "pk", "sk")
	// Put an item
	putItem(t, ddbSvc, "TestTable", map[string]any{
		"pk": map[string]any{"S": "user1"},
		"sk": map[string]any{"S": "profile"},
		"name": map[string]any{"S": "Alice"},
	})
	// Create a DAX cluster
	daxSvc.HandleRequest(jsonCtx("CreateCluster", map[string]any{
		"ClusterName": "test-cluster", "NodeType": "dax.r4.large",
		"ReplicationFactor": 1, "IamRoleArn": "arn:aws:iam::123456789012:role/r",
	}))

	dp := dax.NewDataPlane(daxSvc, ddbSvc)
	return dp, ddbSvc
}

func createTable(t *testing.T, svc *dynamodbsvc.DynamoDBService, name, pk, sk string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"TableName": name,
		"KeySchema": []any{
			map[string]any{"AttributeName": pk, "KeyType": "HASH"},
			map[string]any{"AttributeName": sk, "KeyType": "RANGE"},
		},
		"AttributeDefinitions": []any{
			map[string]any{"AttributeName": pk, "AttributeType": "S"},
			map[string]any{"AttributeName": sk, "AttributeType": "S"},
		},
		"BillingMode": "PAY_PER_REQUEST",
	})
	ctx := &service.RequestContext{
		Action: "CreateTable", Body: body, Region: "us-east-1", AccountID: "123456789012",
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := svc.HandleRequest(ctx)
	require.NoError(t, err)
}

func putItem(t *testing.T, svc *dynamodbsvc.DynamoDBService, table string, item map[string]any) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{"TableName": table, "Item": item})
	ctx := &service.RequestContext{
		Action: "PutItem", Body: body, Region: "us-east-1", AccountID: "123456789012",
		RawRequest: httptest.NewRequest(http.MethodPost, "/", nil),
		Identity:   &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	_, err := svc.HandleRequest(ctx)
	require.NoError(t, err)
}

func TestDataPlane_GetItemReadThrough(t *testing.T) {
	dp, _ := setupDataPlane(t)

	// First call — cache miss, reads from DynamoDB
	reqBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()

	dp.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	item := resp["Item"].(map[string]any)
	assert.Equal(t, map[string]any{"S": "Alice"}, item["name"])

	// Second call — cache hit
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(reqBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()

	dp.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	stats := dp.ClusterStats("test-cluster")
	assert.Equal(t, int64(1), stats.ItemHits)
	assert.Equal(t, int64(1), stats.ItemMisses)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_GetItemReadThrough -v`
Expected: FAIL — `DataPlane` and `NewDataPlane` undefined

- [ ] **Step 3: Implement DataPlane with GetItem handler**

```go
// dataplane.go
package dax

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/Viridian-Inc/cloudmock/pkg/service"
	dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"
)

// DataPlane is an HTTP handler that proxies DynamoDB requests through a DAX cache.
type DataPlane struct {
	daxService *DAXService
	ddbService *dynamodbsvc.DynamoDBService
}

// NewDataPlane creates a new DAX data-plane proxy.
func NewDataPlane(daxSvc *DAXService, ddbSvc *dynamodbsvc.DynamoDBService) *DataPlane {
	return &DataPlane{daxService: daxSvc, ddbService: ddbSvc}
}

// ClusterStats returns the cache stats for a cluster.
func (dp *DataPlane) ClusterStats(clusterName string) CacheStats {
	cache := dp.daxService.store.GetClusterCache(clusterName)
	return cache.Stats()
}

func (dp *DataPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	target := r.Header.Get("X-Amz-Target")
	action := target
	if idx := strings.LastIndex(target, "."); idx >= 0 {
		action = target[idx+1:]
	}

	cluster := r.Header.Get("X-Dax-Cluster")
	if cluster == "" {
		cluster = "default"
	}
	cache := dp.daxService.store.GetClusterCache(cluster)

	switch action {
	case "GetItem":
		dp.handleGetItem(w, body, cache, cluster)
	case "":
		http.Error(w, "missing X-Amz-Target header", http.StatusBadRequest)
	default:
		// For unimplemented actions, pass through to DynamoDB directly
		dp.passThrough(w, action, body)
	}
}

func (dp *DataPlane) handleGetItem(w http.ResponseWriter, body []byte, cache *Cache, cluster string) {
	var req struct {
		TableName string         `json:"TableName"`
		Key       map[string]any `json:"Key"`
	}
	if err := json.Unmarshal(body, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	pk, sk := extractKeyStrings(req.Key)

	// Check cache
	if cached := cache.GetItem(req.TableName, pk, sk); cached != nil {
		writeJSON(w, cached)
		return
	}

	// Cache miss — forward to DynamoDB
	resp := dp.forwardToDynamo("GetItem", body)
	if resp == nil {
		http.Error(w, "DynamoDB request failed", http.StatusInternalServerError)
		return
	}

	// Cache the response
	cache.SetItem(req.TableName, pk, sk, resp)
	writeJSON(w, resp)
}

func (dp *DataPlane) passThrough(w http.ResponseWriter, action string, body []byte) {
	resp := dp.forwardToDynamo(action, body)
	if resp == nil {
		http.Error(w, "DynamoDB request failed", http.StatusInternalServerError)
		return
	}
	writeJSON(w, resp)
}

func (dp *DataPlane) forwardToDynamo(action string, body []byte) any {
	ctx := &service.RequestContext{
		Action:    action,
		Body:      body,
		Region:    "us-east-1",
		AccountID: "123456789012",
		RawRequest: &http.Request{Method: http.MethodPost},
		Identity:  &service.CallerIdentity{AccountID: "123456789012", ARN: "arn:aws:iam::123456789012:root"},
	}
	resp, err := dp.ddbService.HandleRequest(ctx)
	if err != nil {
		return nil
	}
	return resp.Body
}

func extractKeyStrings(key map[string]any) (string, string) {
	pk := ""
	sk := ""
	for _, v := range key {
		if m, ok := v.(map[string]any); ok {
			for _, val := range m {
				if pk == "" {
					pk = fmt.Sprintf("%v", val)
				} else {
					sk = fmt.Sprintf("%v", val)
				}
			}
		}
	}
	return pk, sk
}

func queryHash(body []byte) string {
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h[:16])
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/x-amz-json-1.0")
	json.NewEncoder(w).Encode(v)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_GetItemReadThrough -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/dataplane.go services/dax/dataplane_test.go && git commit -m "feat(dax): add data-plane with GetItem read-through caching"
```

---

### Task 8: Data-Plane — Write-Through with Invalidation

**Files:**
- Modify: `services/dax/dataplane.go`
- Modify: `services/dax/dataplane_test.go`

- [ ] **Step 1: Write failing test for PutItem write-through invalidation**

```go
func TestDataPlane_PutItemInvalidatesCache(t *testing.T) {
	dp, _ := setupDataPlane(t)

	// Read item into cache
	getBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	// Update the item via PutItem
	putBody := `{"TableName":"TestTable","Item":{"pk":{"S":"user1"},"sk":{"S":"profile"},"name":{"S":"Bob"}}}`
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(putBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.PutItem")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()
	dp.ServeHTTP(w2, req2)
	require.Equal(t, http.StatusOK, w2.Code)

	// Read again — should be cache miss (invalidated) and return updated item
	req3 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req3.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req3.Header.Set("X-Dax-Cluster", "test-cluster")
	w3 := httptest.NewRecorder()
	dp.ServeHTTP(w3, req3)

	var resp map[string]any
	require.NoError(t, json.Unmarshal(w3.Body.Bytes(), &resp))
	item := resp["Item"].(map[string]any)
	assert.Equal(t, map[string]any{"S": "Bob"}, item["name"])

	stats := dp.ClusterStats("test-cluster")
	assert.True(t, stats.Invalidations >= 1)
	assert.True(t, stats.WriteThroughs >= 1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_PutItemInvalidatesCache -v`
Expected: FAIL — PutItem not handled

- [ ] **Step 3: Add PutItem, UpdateItem, DeleteItem handlers**

Add to the `ServeHTTP` switch in `dataplane.go`:

```go
	case "PutItem":
		dp.handleWriteThrough(w, action, body, cache, cluster)
	case "UpdateItem":
		dp.handleWriteThrough(w, action, body, cache, cluster)
	case "DeleteItem":
		dp.handleWriteThrough(w, action, body, cache, cluster)
```

Add the handler:

```go
func (dp *DataPlane) handleWriteThrough(w http.ResponseWriter, action string, body []byte, cache *Cache, cluster string) {
	// Forward write to DynamoDB first
	resp := dp.forwardToDynamo(action, body)
	if resp == nil {
		http.Error(w, "DynamoDB write failed", http.StatusInternalServerError)
		return
	}

	// Extract table and key for invalidation
	var req struct {
		TableName string         `json:"TableName"`
		Key       map[string]any `json:"Key"`
		Item      map[string]any `json:"Item"`
	}
	json.Unmarshal(body, &req)

	strategy := dp.daxService.store.GetWriteStrategy(cluster)

	if action == "PutItem" && req.Item != nil {
		pk, sk := extractKeyStrings(req.Item)
		if strategy == "update-cache" {
			cache.SetItem(req.TableName, pk, sk, resp)
		} else {
			cache.InvalidateItem(req.TableName, pk, sk)
		}
	} else if req.Key != nil {
		pk, sk := extractKeyStrings(req.Key)
		cache.InvalidateItem(req.TableName, pk, sk)
	}

	atomic.AddInt64(&cache.stats.WriteThroughs, 1)
	writeJSON(w, resp)
}
```

Add `"sync/atomic"` to `dataplane.go` imports. Make `Cache.stats` exported as `Stats_` or add a method:

```go
// IncrWriteThroughs increments the write-through counter.
func (c *Cache) IncrWriteThroughs() {
	atomic.AddInt64(&c.stats.WriteThroughs, 1)
}
```

Replace `atomic.AddInt64(&cache.stats.WriteThroughs, 1)` with `cache.IncrWriteThroughs()`.

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_PutItem -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/dataplane.go services/dax/dataplane_test.go services/dax/cache.go && git commit -m "feat(dax): add write-through with invalidation for PutItem/UpdateItem/DeleteItem"
```

---

### Task 9: Data-Plane — Query Read-Through

**Files:**
- Modify: `services/dax/dataplane.go`
- Modify: `services/dax/dataplane_test.go`

- [ ] **Step 1: Write failing test for Query caching**

```go
func TestDataPlane_QueryReadThrough(t *testing.T) {
	dp, _ := setupDataPlane(t)

	queryBody := `{"TableName":"TestTable","KeyConditionExpression":"pk = :pk","ExpressionAttributeValues":{":pk":{"S":"user1"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(queryBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.Query")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	// Second call — cache hit
	req2 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(queryBody))
	req2.Header.Set("X-Amz-Target", "DynamoDB_20120810.Query")
	req2.Header.Set("X-Dax-Cluster", "test-cluster")
	w2 := httptest.NewRecorder()
	dp.ServeHTTP(w2, req2)
	assert.Equal(t, http.StatusOK, w2.Code)

	stats := dp.ClusterStats("test-cluster")
	assert.Equal(t, int64(1), stats.QueryHits)
	assert.Equal(t, int64(1), stats.QueryMisses)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_QueryReadThrough -v`
Expected: FAIL — Query not handled

- [ ] **Step 3: Add Query and Scan handlers**

Add to `ServeHTTP` switch:

```go
	case "Query":
		dp.handleQueryReadThrough(w, body, cache)
	case "Scan":
		dp.handleQueryReadThrough(w, body, cache)
```

Add handler:

```go
func (dp *DataPlane) handleQueryReadThrough(w http.ResponseWriter, body []byte, cache *Cache) {
	var req struct {
		TableName string `json:"TableName"`
	}
	json.Unmarshal(body, &req)

	qKey := req.TableName + "|" + queryHash(body)

	if cached := cache.GetQuery(qKey); cached != nil {
		writeJSON(w, cached)
		return
	}

	resp := dp.forwardToDynamo("Query", body)
	if resp == nil {
		http.Error(w, "DynamoDB query failed", http.StatusInternalServerError)
		return
	}

	cache.SetQuery(qKey, resp)
	writeJSON(w, resp)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_QueryReadThrough -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/dataplane.go services/dax/dataplane_test.go && git commit -m "feat(dax): add Query/Scan read-through caching"
```

---

### Task 10: Data-Plane — Stats Endpoint

**Files:**
- Modify: `services/dax/dataplane.go`
- Modify: `services/dax/dataplane_test.go`

- [ ] **Step 1: Write failing test for stats endpoint**

```go
func TestDataPlane_StatsEndpoint(t *testing.T) {
	dp, _ := setupDataPlane(t)

	// Do a GetItem to generate stats
	getBody := `{"TableName":"TestTable","Key":{"pk":{"S":"user1"},"sk":{"S":"profile"}}}`
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(getBody))
	req.Header.Set("X-Amz-Target", "DynamoDB_20120810.GetItem")
	req.Header.Set("X-Dax-Cluster", "test-cluster")
	w := httptest.NewRecorder()
	dp.ServeHTTP(w, req)

	// GET stats
	statsReq := httptest.NewRequest(http.MethodGet, "/stats/test-cluster", nil)
	statsW := httptest.NewRecorder()
	dp.ServeHTTP(statsW, statsReq)

	assert.Equal(t, http.StatusOK, statsW.Code)
	var stats CacheStats
	require.NoError(t, json.Unmarshal(statsW.Body.Bytes(), &stats))
	assert.Equal(t, int64(1), stats.ItemMisses)
	assert.Equal(t, int64(1), stats.ItemSize)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_StatsEndpoint -v`
Expected: FAIL — stats route not matched

- [ ] **Step 3: Add stats route to ServeHTTP**

Update `ServeHTTP` in `dataplane.go` to check for GET /stats/ before the DynamoDB action routing:

```go
func (dp *DataPlane) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Stats endpoint
	if r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/stats/") {
		clusterName := strings.TrimPrefix(r.URL.Path, "/stats/")
		stats := dp.ClusterStats(clusterName)
		writeJSON(w, stats)
		return
	}

	// ... rest of existing ServeHTTP code
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -run TestDataPlane_StatsEndpoint -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
cd /Users/megan/cloudmock && git add services/dax/dataplane.go services/dax/dataplane_test.go && git commit -m "feat(dax): add /stats/{cluster} endpoint for cache metrics"
```

---

### Task 11: Wire Data-Plane into Gateway

**Files:**
- Modify: `cmd/gateway/main.go`

- [ ] **Step 1: Add DAX data-plane server startup after service registration**

Find line 781 in `cmd/gateway/main.go`:

```go
_ = registerOrDefer("dax", func() service.Service { return daxsvc.New(cfg.AccountID, cfg.Region) })
```

Replace with:

```go
var daxService *daxsvc.DAXService
if eagerAll || minimalSet["dax"] {
	daxService = daxsvc.New(cfg.AccountID, cfg.Region)
	registry.Register(daxService)
} else {
	registry.RegisterLazy("dax", func() service.Service {
		daxService = daxsvc.New(cfg.AccountID, cfg.Region)
		return daxService
	})
}
```

Then after the main HTTP server starts (after `ListenAndServe` setup), add:

```go
// DAX Data-Plane server on :8111
go func() {
	daxPort := os.Getenv("CLOUDMOCK_DAX_PORT")
	if daxPort == "" {
		daxPort = "8111"
	}
	if daxService == nil {
		daxService = daxsvc.New(cfg.AccountID, cfg.Region)
	}
	ddbSvc := registry.Get("dynamodb")
	if ddbSvc == nil {
		slog.Warn("DAX data-plane: DynamoDB service not registered, skipping")
		return
	}
	dp := daxsvc.NewDataPlane(daxService, ddbSvc.(*dynamodbsvc.DynamoDBService))
	slog.Info("DAX data-plane listening", "port", daxPort)
	if err := http.ListenAndServe(":"+daxPort, dp); err != nil {
		slog.Error("DAX data-plane server failed", "error", err)
	}
}()
```

Add `dynamodbsvc "github.com/Viridian-Inc/cloudmock/services/dynamodb"` to imports if not present.

- [ ] **Step 2: Build to verify compilation**

Run: `cd /Users/megan/cloudmock && go build ./cmd/gateway/`
Expected: No errors

- [ ] **Step 3: Commit**

```bash
cd /Users/megan/cloudmock && git add cmd/gateway/main.go && git commit -m "feat(dax): wire data-plane HTTP server on :8111 in gateway"
```

---

### Task 12: Update Documentation

**Files:**
- Modify: `website/src/content/docs/docs/services/dax.md`

- [ ] **Step 1: Append data-plane section to dax.md**

Add after the existing "Known Differences from AWS" section:

```markdown
## Data-Plane (Caching Proxy)

CloudMock includes a DAX data-plane HTTP server on port `8111` that acts as a caching proxy for DynamoDB operations. Unlike real DAX (which uses a proprietary binary protocol), CloudMock's data-plane accepts standard DynamoDB JSON requests over HTTP.

### Supported Data-Plane Operations

| Operation | Caching Behavior |
|-----------|-----------------|
| GetItem | Read-through: cache miss forwards to DynamoDB, result cached |
| Query | Read-through: full result set cached by request hash |
| Scan | Read-through: full result set cached by request hash |
| PutItem | Write-through: writes to DynamoDB, then invalidates/updates cache |
| UpdateItem | Write-through: writes to DynamoDB, then invalidates cache |
| DeleteItem | Write-through: writes to DynamoDB, then invalidates cache |
| BatchGetItem | Pass-through (forwarded to DynamoDB) |
| BatchWriteItem | Pass-through with table invalidation |
| TransactGetItems | Pass-through |
| TransactWriteItems | Pass-through with table invalidation |

### Quick Start

Point your standard DynamoDB SDK at port `8111` instead of `4566`:

```typescript
import { DynamoDBClient, GetItemCommand } from '@aws-sdk/client-dynamodb';

const client = new DynamoDBClient({
  endpoint: 'http://localhost:8111',
  region: 'us-east-1',
  credentials: { accessKeyId: 'test', secretAccessKey: 'test' },
});

// First call: cache miss -> reads from DynamoDB -> caches result
// Second call: cache hit -> returns from cache
const result = await client.send(new GetItemCommand({
  TableName: 'users',
  Key: { pk: { S: 'user1' } },
}));
```

### Cluster Selection

Use the `X-Dax-Cluster` header to route requests to a specific cluster's cache:

```bash
curl -X POST http://localhost:8111 \
  -H 'X-Amz-Target: DynamoDB_20120810.GetItem' \
  -H 'X-Dax-Cluster: my-cluster' \
  -d '{"TableName":"users","Key":{"pk":{"S":"user1"}}}'
```

If omitted, a default cache with default TTLs (5 minutes) is used.

### Cache Configuration

Cache behavior is controlled through DAX parameter groups (via the control-plane API on port `4566`):

| Parameter | Default | Description |
|-----------|---------|-------------|
| `record-ttl-millis` | 300000 | TTL for cached items (5 minutes) |
| `query-ttl-millis` | 300000 | TTL for cached query results |
| `write-strategy` | `invalidate` | `invalidate` (default, matches AWS) or `update-cache` |
| `max-cache-size` | 10000 | Maximum items per cluster cache |

### Cache Statistics

```bash
curl http://localhost:8111/stats/my-cluster
```

Returns:

```json
{
  "itemHits": 1520,
  "itemMisses": 340,
  "queryHits": 200,
  "queryMisses": 80,
  "itemSize": 890,
  "querySize": 45,
  "evictions": 12,
  "writeThroughs": 150,
  "invalidations": 150
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CLOUDMOCK_DAX_PORT` | `8111` | Port for the DAX data-plane server |
```

- [ ] **Step 2: Commit**

```bash
cd /Users/megan/cloudmock && git add website/src/content/docs/docs/services/dax.md && git commit -m "docs(dax): document data-plane caching proxy"
```

---

### Task 13: Run Full Test Suite

- [ ] **Step 1: Run all DAX tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dax/ -v -count=1`
Expected: ALL PASS (control plane + cache + data-plane tests)

- [ ] **Step 2: Run existing tests to verify no regressions**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -v -count=1 -timeout 120s`
Expected: ALL PASS

- [ ] **Step 3: Build gateway**

Run: `cd /Users/megan/cloudmock && go build ./cmd/gateway/`
Expected: No errors
