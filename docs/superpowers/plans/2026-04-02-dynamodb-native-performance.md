# DynamoDB Native Performance Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rewrite DynamoDB storage internals to achieve <0.1ms P50 GetItem and O(1)/O(log n) scaling to 10M+ items under 10K concurrent goroutines.

**Architecture:** Replace flat `[]Item` slice with hash-indexed partitions containing B-tree sorted items. Compile expressions into cached ASTs. Replace TTL full-scan with min-heap timer. Per-table locking instead of per-store.

**Tech Stack:** Go 1.26, `github.com/tidwall/btree` (generic B-tree), `container/heap` (stdlib min-heap)

---

## File Structure

```
pkg/collections/
├── ringbuffer.go          # Generic ring buffer (shared with SQS plan)
├── ringbuffer_test.go
├── minheap.go             # Generic min-heap (shared with SQS plan)
└── minheap_test.go

services/dynamodb/
├── service.go             # KEEP — minimal wiring changes only
├── handlers.go            # KEEP — minimal changes (pass new params)
├── store.go               # REWRITE — per-table locks, delegate to table methods
├── table.go               # REWRITE — partition map + B-tree + pkIndex
├── partition.go            # NEW — Partition struct with B-tree
├── expression.go          # REWRITE — AST compiler with cache
├── expression_ast.go      # NEW — AST node types
├── ttl.go                 # REWRITE — heap-based reaper
├── streams.go             # KEEP — unchanged
├── store_test.go          # NEW — storage engine tests
├── table_test.go          # NEW — table + partition tests
├── expression_test.go     # NEW — expression compiler tests
├── ttl_test.go            # NEW — TTL heap tests
├── bench_test.go          # NEW — Go benchmarks at 1K/100K/1M/10M items
└── service_test.go        # KEEP — existing tests must still pass
```

---

### Task 1: Generic Min-Heap

**Files:**
- Create: `pkg/collections/minheap.go`
- Create: `pkg/collections/minheap_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/collections/minheap_test.go`:

```go
package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMinHeap_PushPop(t *testing.T) {
	h := NewMinHeap[int, int](func(a, b int) bool { return a < b })
	h.Push(3, 30)
	h.Push(1, 10)
	h.Push(2, 20)

	key, val, ok := h.Pop()
	require.True(t, ok)
	assert.Equal(t, 1, key)
	assert.Equal(t, 10, val)

	key, val, ok = h.Pop()
	require.True(t, ok)
	assert.Equal(t, 2, key)
	assert.Equal(t, 20, val)
}

func TestMinHeap_Peek(t *testing.T) {
	h := NewMinHeap[int, string](func(a, b int) bool { return a < b })
	_, _, ok := h.Peek()
	assert.False(t, ok)

	h.Push(5, "five")
	h.Push(2, "two")

	key, val, ok := h.Peek()
	require.True(t, ok)
	assert.Equal(t, 2, key)
	assert.Equal(t, "two", val)
	assert.Equal(t, 2, h.Len()) // Peek doesn't remove
}

func TestMinHeap_Empty(t *testing.T) {
	h := NewMinHeap[int, int](func(a, b int) bool { return a < b })
	_, _, ok := h.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, h.Len())
}

func TestMinHeap_Remove(t *testing.T) {
	h := NewMinHeap[int, string](func(a, b int) bool { return a < b })
	h.Push(3, "three")
	h.Push(1, "one")
	h.Push(2, "two")

	removed := h.RemoveByValue("two")
	assert.True(t, removed)
	assert.Equal(t, 2, h.Len())

	key, val, _ := h.Pop()
	assert.Equal(t, 1, key)
	assert.Equal(t, "one", val)

	key, val, _ = h.Pop()
	assert.Equal(t, 3, key)
	assert.Equal(t, "three", val)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./pkg/collections/ -run TestMinHeap -v`
Expected: FAIL — package not found

- [ ] **Step 3: Implement MinHeap**

Create `pkg/collections/minheap.go`:

```go
package collections

import "container/heap"

// MinHeap is a generic min-heap ordered by a comparable key.
type MinHeap[K any, V comparable] struct {
	items  *heapItems[K, V]
	lessFn func(a, b K) bool
}

type heapItem[K any, V comparable] struct {
	key   K
	value V
	index int
}

type heapItems[K any, V comparable] struct {
	data   []*heapItem[K, V]
	lessFn func(a, b K) bool
}

func (h *heapItems[K, V]) Len() int           { return len(h.data) }
func (h *heapItems[K, V]) Less(i, j int) bool { return h.lessFn(h.data[i].key, h.data[j].key) }
func (h *heapItems[K, V]) Swap(i, j int) {
	h.data[i], h.data[j] = h.data[j], h.data[i]
	h.data[i].index = i
	h.data[j].index = j
}

func (h *heapItems[K, V]) Push(x any) {
	item := x.(*heapItem[K, V])
	item.index = len(h.data)
	h.data = append(h.data, item)
}

func (h *heapItems[K, V]) Pop() any {
	old := h.data
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	item.index = -1
	h.data = old[:n-1]
	return item
}

// NewMinHeap creates a min-heap with the given comparison function.
func NewMinHeap[K any, V comparable](less func(a, b K) bool) *MinHeap[K, V] {
	hi := &heapItems[K, V]{lessFn: less}
	heap.Init(hi)
	return &MinHeap[K, V]{items: hi, lessFn: less}
}

// Push adds a key-value pair to the heap.
func (h *MinHeap[K, V]) Push(key K, value V) {
	heap.Push(h.items, &heapItem[K, V]{key: key, value: value})
}

// Pop removes and returns the minimum key-value pair.
func (h *MinHeap[K, V]) Pop() (K, V, bool) {
	if h.items.Len() == 0 {
		var zk K
		var zv V
		return zk, zv, false
	}
	item := heap.Pop(h.items).(*heapItem[K, V])
	return item.key, item.value, true
}

// Peek returns the minimum key-value pair without removing it.
func (h *MinHeap[K, V]) Peek() (K, V, bool) {
	if h.items.Len() == 0 {
		var zk K
		var zv V
		return zk, zv, false
	}
	item := h.items.data[0]
	return item.key, item.value, true
}

// Len returns the number of items.
func (h *MinHeap[K, V]) Len() int {
	return h.items.Len()
}

// RemoveByValue removes the first item with the given value. Returns true if found.
func (h *MinHeap[K, V]) RemoveByValue(value V) bool {
	for i, item := range h.items.data {
		if item.value == value {
			heap.Remove(h.items, i)
			return true
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/megan/cloudmock && go test ./pkg/collections/ -run TestMinHeap -v`
Expected: All 4 tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/collections/minheap.go pkg/collections/minheap_test.go
git commit -m "feat: add generic min-heap to collections package"
```

---

### Task 2: Generic Ring Buffer

**Files:**
- Create: `pkg/collections/ringbuffer.go`
- Create: `pkg/collections/ringbuffer_test.go`

- [ ] **Step 1: Write the failing test**

Create `pkg/collections/ringbuffer_test.go`:

```go
package collections

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRingBuffer_PushPop(t *testing.T) {
	rb := NewRingBuffer[int](4)

	rb.Push(1)
	rb.Push(2)
	rb.Push(3)

	val, ok := rb.Pop()
	require.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = rb.Pop()
	require.True(t, ok)
	assert.Equal(t, 2, val)

	assert.Equal(t, 1, rb.Len())
}

func TestRingBuffer_Empty(t *testing.T) {
	rb := NewRingBuffer[string](4)
	_, ok := rb.Pop()
	assert.False(t, ok)
	assert.Equal(t, 0, rb.Len())
}

func TestRingBuffer_Grow(t *testing.T) {
	rb := NewRingBuffer[int](2)
	for i := 0; i < 100; i++ {
		rb.Push(i)
	}
	assert.Equal(t, 100, rb.Len())

	for i := 0; i < 100; i++ {
		val, ok := rb.Pop()
		require.True(t, ok)
		assert.Equal(t, i, val)
	}
	assert.Equal(t, 0, rb.Len())
}

func TestRingBuffer_Wraparound(t *testing.T) {
	rb := NewRingBuffer[int](4)
	// Fill and drain partially to force wraparound
	rb.Push(1)
	rb.Push(2)
	rb.Pop() // pop 1
	rb.Pop() // pop 2
	// Now head and tail are at index 2
	rb.Push(3)
	rb.Push(4)
	rb.Push(5)
	rb.Push(6) // should wrap around

	assert.Equal(t, 4, rb.Len())
	val, _ := rb.Pop()
	assert.Equal(t, 3, val)
	val, _ = rb.Pop()
	assert.Equal(t, 4, val)
	val, _ = rb.Pop()
	assert.Equal(t, 5, val)
	val, _ = rb.Pop()
	assert.Equal(t, 6, val)
}

func BenchmarkRingBuffer_PushPop(b *testing.B) {
	rb := NewRingBuffer[int](1024)
	for i := 0; i < b.N; i++ {
		rb.Push(i)
		rb.Pop()
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./pkg/collections/ -run TestRingBuffer -v`
Expected: FAIL — `NewRingBuffer` not defined

- [ ] **Step 3: Implement RingBuffer**

Create `pkg/collections/ringbuffer.go`:

```go
package collections

// RingBuffer is a generic FIFO queue backed by a circular array.
type RingBuffer[T any] struct {
	buf  []T
	head int
	tail int
	len  int
	cap  int
}

// NewRingBuffer creates a ring buffer with the given initial capacity.
func NewRingBuffer[T any](initialCap int) *RingBuffer[T] {
	if initialCap < 4 {
		initialCap = 4
	}
	return &RingBuffer[T]{
		buf: make([]T, initialCap),
		cap: initialCap,
	}
}

// Push appends an item to the tail. O(1) amortized.
func (rb *RingBuffer[T]) Push(item T) {
	if rb.len == rb.cap {
		rb.grow()
	}
	rb.buf[rb.tail] = item
	rb.tail = (rb.tail + 1) % rb.cap
	rb.len++
}

// Pop removes and returns the item at the head. O(1).
func (rb *RingBuffer[T]) Pop() (T, bool) {
	if rb.len == 0 {
		var zero T
		return zero, false
	}
	item := rb.buf[rb.head]
	var zero T
	rb.buf[rb.head] = zero // help GC
	rb.head = (rb.head + 1) % rb.cap
	rb.len--
	return item, true
}

// Len returns the number of items in the buffer.
func (rb *RingBuffer[T]) Len() int {
	return rb.len
}

func (rb *RingBuffer[T]) grow() {
	newCap := rb.cap * 2
	newBuf := make([]T, newCap)
	// Copy head..end, then start..tail
	if rb.head < rb.tail {
		copy(newBuf, rb.buf[rb.head:rb.tail])
	} else {
		n := copy(newBuf, rb.buf[rb.head:])
		copy(newBuf[n:], rb.buf[:rb.tail])
	}
	rb.buf = newBuf
	rb.head = 0
	rb.tail = rb.len
	rb.cap = newCap
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd /Users/megan/cloudmock && go test ./pkg/collections/ -v`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/collections/ringbuffer.go pkg/collections/ringbuffer_test.go
git commit -m "feat: add generic ring buffer to collections package"
```

---

### Task 3: DynamoDB Partition and Table Rewrite

**Files:**
- Create: `services/dynamodb/partition.go`
- Rewrite: `services/dynamodb/table.go`
- Create: `services/dynamodb/table_test.go`

This is the core data structure change. Replace `[]Item` with hash-indexed partitions containing B-tree sorted items.

- [ ] **Step 1: Write the failing test**

Create `services/dynamodb/table_test.go`:

```go
package dynamodb

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTable(withSortKey bool) *Table {
	ks := []KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}}
	ad := []AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}}
	if withSortKey {
		ks = append(ks, KeySchemaElement{AttributeName: "sk", KeyType: "RANGE"})
		ad = append(ad, AttributeDefinition{AttributeName: "sk", AttributeType: "S"})
	}
	t := newTable("test-table", ks, ad, "PAY_PER_REQUEST", nil, nil, nil, nil, "123456789012", "us-east-1")
	return t
}

func TestTable_PutAndGet_HashOnly(t *testing.T) {
	tbl := makeTable(false)
	item := Item{"pk": {"S": "user-1"}, "name": {"S": "Alice"}}

	tbl.putItem(item)

	got, found := tbl.getItem(Item{"pk": {"S": "user-1"}})
	require.True(t, found)
	assert.Equal(t, "Alice", got["name"]["S"])
}

func TestTable_PutAndGet_Composite(t *testing.T) {
	tbl := makeTable(true)
	tbl.putItem(Item{"pk": {"S": "user-1"}, "sk": {"S": "profile"}, "name": {"S": "Alice"}})
	tbl.putItem(Item{"pk": {"S": "user-1"}, "sk": {"S": "settings"}, "theme": {"S": "dark"}})

	got, found := tbl.getItem(Item{"pk": {"S": "user-1"}, "sk": {"S": "settings"}})
	require.True(t, found)
	assert.Equal(t, "dark", got["theme"]["S"])
}

func TestTable_PutOverwrite(t *testing.T) {
	tbl := makeTable(false)
	tbl.putItem(Item{"pk": {"S": "user-1"}, "v": {"N": "1"}})
	tbl.putItem(Item{"pk": {"S": "user-1"}, "v": {"N": "2"}})

	got, found := tbl.getItem(Item{"pk": {"S": "user-1"}})
	require.True(t, found)
	assert.Equal(t, "2", got["v"]["N"])
	assert.Equal(t, int64(1), tbl.itemCount())
}

func TestTable_Delete(t *testing.T) {
	tbl := makeTable(false)
	tbl.putItem(Item{"pk": {"S": "user-1"}, "name": {"S": "Alice"}})

	old := tbl.deleteItem(Item{"pk": {"S": "user-1"}})
	assert.NotNil(t, old)
	assert.Equal(t, "Alice", old["name"]["S"])

	_, found := tbl.getItem(Item{"pk": {"S": "user-1"}})
	assert.False(t, found)
	assert.Equal(t, int64(0), tbl.itemCount())
}

func TestTable_QueryPartition(t *testing.T) {
	tbl := makeTable(true)
	for i := 0; i < 100; i++ {
		tbl.putItem(Item{
			"pk":   {"S": "user-1"},
			"sk":   {"S": fmt.Sprintf("item-%03d", i)},
			"data": {"S": fmt.Sprintf("value-%d", i)},
		})
	}

	// Query all items in partition
	items := tbl.queryPartition("user-1", nil, true, 0)
	assert.Equal(t, 100, len(items))
	assert.Equal(t, "item-000", items[0]["sk"]["S"])   // sorted ascending
	assert.Equal(t, "item-099", items[99]["sk"]["S"])
}

func TestTable_QueryPartition_Descending(t *testing.T) {
	tbl := makeTable(true)
	for i := 0; i < 10; i++ {
		tbl.putItem(Item{
			"pk": {"S": "user-1"},
			"sk": {"S": fmt.Sprintf("item-%03d", i)},
		})
	}

	items := tbl.queryPartition("user-1", nil, false, 5)
	assert.Equal(t, 5, len(items))
	assert.Equal(t, "item-009", items[0]["sk"]["S"]) // descending
}

func TestTable_ItemCount(t *testing.T) {
	tbl := makeTable(false)
	assert.Equal(t, int64(0), tbl.itemCount())

	for i := 0; i < 1000; i++ {
		tbl.putItem(Item{"pk": {"S": fmt.Sprintf("key-%d", i)}})
	}
	assert.Equal(t, int64(1000), tbl.itemCount())
}

func BenchmarkTable_GetItem_1M(b *testing.B) {
	tbl := makeTable(false)
	for i := 0; i < 1_000_000; i++ {
		tbl.putItem(Item{"pk": {"S": fmt.Sprintf("key-%d", i)}})
	}
	key := Item{"pk": {"S": "key-500000"}}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.getItem(key)
	}
}

func BenchmarkTable_PutItem(b *testing.B) {
	tbl := makeTable(false)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.putItem(Item{"pk": {"S": fmt.Sprintf("key-%d", i)}})
	}
}

func BenchmarkTable_QueryPartition_100(b *testing.B) {
	tbl := makeTable(true)
	for i := 0; i < 10000; i++ {
		tbl.putItem(Item{
			"pk": {"S": "user-1"},
			"sk": {"S": fmt.Sprintf("item-%05d", i)},
		})
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tbl.queryPartition("user-1", nil, true, 100)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -run TestTable_ -v`
Expected: FAIL — `newTable` signature doesn't match, methods don't exist

- [ ] **Step 3: Add tidwall/btree dependency**

Run: `cd /Users/megan/cloudmock && go get github.com/tidwall/btree@latest`

- [ ] **Step 4: Create partition.go**

Create `services/dynamodb/partition.go`:

```go
package dynamodb

import (
	"strings"

	"github.com/tidwall/btree"
)

// Partition holds all items sharing the same partition key.
// For tables with a sort key, items are stored in a B-tree sorted by sort key.
// For tables without a sort key, there is at most one item.
type Partition struct {
	items *btree.BTreeG[Item] // sorted by sort key; nil if no sort key
	item  Item                // single item for hash-only tables
	count int
}

// newPartition creates a partition. If sortKeyName is non-empty, uses B-tree storage.
func newPartition(sortKeyName string) *Partition {
	p := &Partition{}
	if sortKeyName != "" {
		p.items = btree.NewBTreeG[Item](func(a, b Item) bool {
			return sortKeyValue(a, sortKeyName) < sortKeyValue(b, sortKeyName)
		})
	}
	return p
}

// put inserts or overwrites an item. Returns the old item if overwritten, and whether it was an overwrite.
func (p *Partition) put(item Item, sortKeyName string) (Item, bool) {
	if p.items == nil {
		// Hash-only table
		old := p.item
		wasOverwrite := p.count > 0
		p.item = item
		if !wasOverwrite {
			p.count = 1
		}
		return old, wasOverwrite
	}

	// Composite key — B-tree
	old, replaced := p.items.Set(item)
	if !replaced {
		p.count++
	}
	return old, replaced
}

// get retrieves an item by sort key. For hash-only, sortKey is ignored.
func (p *Partition) get(sortKey Item, sortKeyName string) (Item, bool) {
	if p.items == nil {
		if p.count == 0 {
			return nil, false
		}
		return p.item, true
	}

	// Build a search key with just the sort key field
	found, ok := p.items.Get(sortKey)
	return found, ok
}

// delete removes an item by sort key. Returns the deleted item.
func (p *Partition) delete(sortKey Item, sortKeyName string) (Item, bool) {
	if p.items == nil {
		if p.count == 0 {
			return nil, false
		}
		old := p.item
		p.item = nil
		p.count = 0
		return old, true
	}

	old, deleted := p.items.Delete(sortKey)
	if deleted {
		p.count--
	}
	return old, deleted
}

// scan iterates all items in sort key order. If desc is true, iterates in reverse.
// If limit > 0, stops after limit items.
func (p *Partition) scan(ascending bool, limit int) []Item {
	if p.items == nil {
		if p.count == 0 {
			return nil
		}
		return []Item{p.item}
	}

	var result []Item
	iter := func(item Item) bool {
		result = append(result, item)
		return limit <= 0 || len(result) < limit
	}

	if ascending {
		p.items.Ascend(Item{}, iter)
	} else {
		p.items.Descend(Item{}, iter)
	}
	return result
}

// sortKeyValue extracts the sort key's string representation for ordering.
func sortKeyValue(item Item, sortKeyName string) string {
	av, ok := item[sortKeyName]
	if !ok {
		return ""
	}
	if s, ok := av["S"].(string); ok {
		return s
	}
	if n, ok := av["N"].(string); ok {
		return padNumber(n)
	}
	return ""
}

// padNumber pads a numeric string for correct string-order comparison.
// Handles negative numbers and decimal points.
func padNumber(n string) string {
	if strings.HasPrefix(n, "-") {
		// For negative numbers, invert for correct ordering
		return "0" + n
	}
	// Pad positive numbers with leading zeros for uniform length
	return "1" + strings.Repeat("0", 20-len(n)) + n
}
```

- [ ] **Step 5: Rewrite table.go**

Rewrite `services/dynamodb/table.go` to use partitions and a primary key index. Keep all existing exported type definitions (`AttributeValue`, `Item`, `KeySchemaElement`, `AttributeDefinition`, `ProvisionedThroughput`, `GSI`, `LSI`, `Table`) but change the Table struct internals:

```go
package dynamodb

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

// Type aliases — these are the same as before
type AttributeValue = map[string]any
type Item = map[string]AttributeValue

type KeySchemaElement struct {
	AttributeName string
	KeyType       string
}

type AttributeDefinition struct {
	AttributeName string
	AttributeType string
}

type ProvisionedThroughput struct {
	ReadCapacityUnits  int64
	WriteCapacityUnits int64
}

type GSI struct {
	IndexName            string
	KeySchema            []KeySchemaElement
	Projection           map[string]any
	ProvisionedThroughput *ProvisionedThroughput
}

type LSI struct {
	IndexName  string
	KeySchema  []KeySchemaElement
	Projection map[string]any
}

// Table holds all data for a single DynamoDB table.
type Table struct {
	Name                  string
	ARN                   string
	KeySchema             []KeySchemaElement
	AttributeDefinitions  []AttributeDefinition
	Status                string
	CreationDateTime      float64
	BillingMode           string
	ProvisionedThroughput *ProvisionedThroughput
	GSIs                  []GSI
	LSIs                  []LSI
	Stream                *Stream
	TTL                   *TTLSpecification

	// Internal storage — not exported
	partitions map[string]*Partition // partitionKey value -> partition
	pkIndex    map[string]Item       // serialized(pk+sk) -> item reference
	gsiStores  map[string]*IndexStore
	lsiStores  map[string]*IndexStore
	count      atomic.Int64

	hashKeyName  string
	rangeKeyName string
}

// IndexStore holds items for a single GSI or LSI.
type IndexStore struct {
	partitions map[string]*Partition
	hashKey    string
	rangeKey   string
}

func newTable(name string, ks []KeySchemaElement, ad []AttributeDefinition, billing string, pt *ProvisionedThroughput, gsis []GSI, lsis []LSI, streamSpec *StreamSpecification, accountID, region string) *Table {
	t := &Table{
		Name:                  name,
		ARN:                   fmt.Sprintf("arn:aws:dynamodb:%s:%s:table/%s", region, accountID, name),
		KeySchema:             ks,
		AttributeDefinitions:  ad,
		Status:                "ACTIVE",
		CreationDateTime:      float64(time.Now().Unix()),
		BillingMode:           billing,
		ProvisionedThroughput: pt,
		GSIs:                  gsis,
		LSIs:                  lsis,
		partitions:            make(map[string]*Partition),
		pkIndex:               make(map[string]Item),
		gsiStores:             make(map[string]*IndexStore),
		lsiStores:             make(map[string]*IndexStore),
	}

	for _, k := range ks {
		if k.KeyType == "HASH" {
			t.hashKeyName = k.AttributeName
		} else if k.KeyType == "RANGE" {
			t.rangeKeyName = k.AttributeName
		}
	}

	for _, gsi := range gsis {
		is := &IndexStore{partitions: make(map[string]*Partition)}
		for _, k := range gsi.KeySchema {
			if k.KeyType == "HASH" {
				is.hashKey = k.AttributeName
			} else if k.KeyType == "RANGE" {
				is.rangeKey = k.AttributeName
			}
		}
		t.gsiStores[gsi.IndexName] = is
	}

	for _, lsi := range lsis {
		is := &IndexStore{partitions: make(map[string]*Partition)}
		for _, k := range lsi.KeySchema {
			if k.KeyType == "HASH" {
				is.hashKey = k.AttributeName
			} else if k.KeyType == "RANGE" {
				is.rangeKey = k.AttributeName
			}
		}
		t.lsiStores[lsi.IndexName] = is
	}

	if streamSpec != nil && streamSpec.StreamEnabled {
		t.Stream = newStream(t.ARN, name, streamSpec.StreamViewType)
	}

	return t
}

// serializeKey creates a unique string key for the primary key index.
func (t *Table) serializeKey(item Item) string {
	pk := attrString(item[t.hashKeyName])
	if t.rangeKeyName == "" {
		return pk
	}
	return pk + "\x00" + attrString(item[t.rangeKeyName])
}

func attrString(av AttributeValue) string {
	if av == nil {
		return ""
	}
	if s, ok := av["S"].(string); ok {
		return "S:" + s
	}
	if n, ok := av["N"].(string); ok {
		return "N:" + n
	}
	if b, ok := av["B"].(string); ok {
		return "B:" + b
	}
	return fmt.Sprintf("%v", av)
}

// putItem inserts or overwrites an item. Returns the old item if overwritten.
func (t *Table) putItem(item Item) Item {
	pkVal := attrString(item[t.hashKeyName])
	key := t.serializeKey(item)

	// Get or create partition
	p, ok := t.partitions[pkVal]
	if !ok {
		p = newPartition(t.rangeKeyName)
		t.partitions[pkVal] = p
	}

	old, replaced := p.put(item, t.rangeKeyName)
	t.pkIndex[key] = item

	if replaced {
		// Deindex old item from GSIs/LSIs
		t.deindexItem(old)
	} else {
		t.count.Add(1)
	}

	// Index new item in GSIs/LSIs
	t.indexItem(item)

	return old
}

// getItem retrieves an item by primary key. O(1).
func (t *Table) getItem(key Item) (Item, bool) {
	serialized := t.serializeKey(key)
	item, ok := t.pkIndex[serialized]
	return item, ok
}

// deleteItem removes an item by primary key. Returns the old item.
func (t *Table) deleteItem(key Item) Item {
	serialized := t.serializeKey(key)
	item, ok := t.pkIndex[serialized]
	if !ok {
		return nil
	}

	delete(t.pkIndex, serialized)

	pkVal := attrString(key[t.hashKeyName])
	if p, exists := t.partitions[pkVal]; exists {
		p.delete(key, t.rangeKeyName)
		if p.count == 0 {
			delete(t.partitions, pkVal)
		}
	}

	t.count.Add(-1)
	t.deindexItem(item)
	return item
}

// queryPartition returns all items in a partition, sorted by sort key.
func (t *Table) queryPartition(partitionKeyValue string, startKey Item, ascending bool, limit int) []Item {
	p, ok := t.partitions[partitionKeyValue]
	if !ok {
		return nil
	}
	return p.scan(ascending, limit)
}

// scanAll iterates all items across all partitions.
func (t *Table) scanAll(limit int) []Item {
	var result []Item
	for _, p := range t.partitions {
		items := p.scan(true, 0)
		result = append(result, items...)
		if limit > 0 && len(result) >= limit {
			return result[:limit]
		}
	}
	return result
}

// itemCount returns the total number of items.
func (t *Table) itemCount() int64 {
	return t.count.Load()
}

// indexItem adds an item to all GSI/LSI stores.
func (t *Table) indexItem(item Item) {
	for _, gsiStore := range t.gsiStores {
		pkAV, hasPK := item[gsiStore.hashKey]
		if !hasPK {
			continue
		}
		pkVal := attrString(pkAV)
		p, ok := gsiStore.partitions[pkVal]
		if !ok {
			p = newPartition(gsiStore.rangeKey)
			gsiStore.partitions[pkVal] = p
		}
		p.put(item, gsiStore.rangeKey)
	}

	for _, lsiStore := range t.lsiStores {
		pkAV, hasPK := item[lsiStore.hashKey]
		if !hasPK {
			continue
		}
		pkVal := attrString(pkAV)
		p, ok := lsiStore.partitions[pkVal]
		if !ok {
			p = newPartition(lsiStore.rangeKey)
			lsiStore.partitions[pkVal] = p
		}
		p.put(item, lsiStore.rangeKey)
	}
}

// deindexItem removes an item from all GSI/LSI stores.
func (t *Table) deindexItem(item Item) {
	for _, gsiStore := range t.gsiStores {
		pkAV, hasPK := item[gsiStore.hashKey]
		if !hasPK {
			continue
		}
		pkVal := attrString(pkAV)
		if p, ok := gsiStore.partitions[pkVal]; ok {
			p.delete(item, gsiStore.rangeKey)
		}
	}

	for _, lsiStore := range t.lsiStores {
		pkAV, hasPK := item[lsiStore.hashKey]
		if !hasPK {
			continue
		}
		pkVal := attrString(pkAV)
		if p, ok := lsiStore.partitions[pkVal]; ok {
			p.delete(item, lsiStore.rangeKey)
		}
	}
}

// Backward compatibility helpers used by existing code
func (t *Table) hashKey() string  { return t.hashKeyName }
func (t *Table) rangeKey() string { return t.rangeKeyName }

func (t *Table) keyMatchesItem(key Item, item Item) bool {
	return t.serializeKey(key) == t.serializeKey(item)
}

func copyItem(item Item) Item {
	if item == nil {
		return nil
	}
	cp := make(Item, len(item))
	for k, v := range item {
		cp[k] = v
	}
	return cp
}

func gsiHashKeyName(ks []KeySchemaElement) string {
	for _, k := range ks {
		if k.KeyType == "HASH" {
			return k.AttributeName
		}
	}
	return ""
}

func gsiRangeKeyName(ks []KeySchemaElement) string {
	for _, k := range ks {
		if k.KeyType == "RANGE" {
			return k.AttributeName
		}
	}
	return ""
}

func avEqual(a, b AttributeValue) bool {
	return attrString(a) == attrString(b)
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -run TestTable_ -v`
Expected: All table tests PASS

- [ ] **Step 7: Run existing tests to verify no regressions**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -v`
Expected: Existing tests should still compile (may need minor store.go fixes in next task)

- [ ] **Step 8: Commit**

```bash
git add services/dynamodb/partition.go services/dynamodb/table.go services/dynamodb/table_test.go
git commit -m "feat(dynamodb): rewrite table storage with partitions and B-tree indexes"
```

---

### Task 4: DynamoDB Store Rewrite (Per-Table Locking)

**Files:**
- Rewrite: `services/dynamodb/store.go`
- Create: `services/dynamodb/store_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/dynamodb/store_test.go`:

```go
package dynamodb

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStore_CreateAndDescribe(t *testing.T) {
	s := NewTableStore("123456789012", "us-east-1")
	tbl, err := s.CreateTable("users",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)
	require.Nil(t, err)
	assert.Equal(t, "users", tbl.Name)

	desc, err := s.DescribeTable("users")
	require.Nil(t, err)
	assert.Equal(t, "ACTIVE", desc.Status)
}

func TestStore_PutGetDelete(t *testing.T) {
	s := NewTableStore("123456789012", "us-east-1")
	s.CreateTable("users",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)

	err := s.PutItem("users", Item{"pk": {"S": "u1"}, "name": {"S": "Alice"}})
	require.Nil(t, err)

	item, err := s.GetItem("users", Item{"pk": {"S": "u1"}}, "", nil)
	require.Nil(t, err)
	assert.Equal(t, "Alice", item["name"]["S"])

	err = s.DeleteItem("users", Item{"pk": {"S": "u1"}})
	require.Nil(t, err)

	item, err = s.GetItem("users", Item{"pk": {"S": "u1"}}, "", nil)
	require.Nil(t, err)
	assert.Nil(t, item)
}

func TestStore_ConcurrentAccess(t *testing.T) {
	s := NewTableStore("123456789012", "us-east-1")
	s.CreateTable("counters",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 1000; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			s.PutItem("counters", Item{"pk": {"S": fmt.Sprintf("key-%d", n)}})
		}(i)
	}
	wg.Wait()

	// All 1000 items should be present
	desc, _ := s.DescribeTable("counters")
	assert.Equal(t, int64(1000), desc.itemCount())
}

func BenchmarkStore_GetItem_Concurrent(b *testing.B) {
	s := NewTableStore("123456789012", "us-east-1")
	s.CreateTable("bench",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)

	for i := 0; i < 100000; i++ {
		s.PutItem("bench", Item{"pk": {"S": fmt.Sprintf("key-%d", i)}})
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			s.GetItem("bench", Item{"pk": {"S": fmt.Sprintf("key-%d", i%100000)}}, "", nil)
			i++
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -run TestStore_ -v`
Expected: FAIL or compile errors from store.go mismatch

- [ ] **Step 3: Rewrite store.go with per-table locking**

Rewrite `services/dynamodb/store.go`. Key changes:
- Replace single `sync.RWMutex` with store-level lock (for table create/delete/list only) + per-table `sync.RWMutex`
- Delegate item operations to `table.putItem()`, `table.getItem()`, `table.deleteItem()` which are now O(1)
- Query/Scan delegate to `table.queryPartition()` and `table.scanAll()` with expression filtering
- Keep all existing exported method signatures identical

The store.go rewrite keeps the same public API but internally:
- `getTable()` acquires store RLock to find table, then releases it
- `PutItem`/`GetItem`/`DeleteItem` acquire table-level lock
- `Query`/`Scan` acquire table-level RLock
- `CreateTable`/`DeleteTable` acquire store-level Lock

- [ ] **Step 4: Run all tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -v`
Expected: All tests PASS (both new store tests and existing service tests)

- [ ] **Step 5: Run benchmarks**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -bench=. -benchmem`
Expected: GetItem at 100K items should be <100ns/op

- [ ] **Step 6: Commit**

```bash
git add services/dynamodb/store.go services/dynamodb/store_test.go
git commit -m "feat(dynamodb): rewrite store with per-table locking and O(1) lookups"
```

---

### Task 5: Expression AST Compiler

**Files:**
- Create: `services/dynamodb/expression_ast.go`
- Rewrite: `services/dynamodb/expression.go`
- Create: `services/dynamodb/expression_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/dynamodb/expression_test.go`:

```go
package dynamodb

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExpr_CompileAndEval_Equals(t *testing.T) {
	expr := CompileCondition("#pk = :val", map[string]string{"#pk": "userId"}, map[string]AttributeValue{":val": {"S": "u1"}})
	item := Item{"userId": {"S": "u1"}, "name": {"S": "Alice"}}
	assert.True(t, expr.Evaluate(item))

	item2 := Item{"userId": {"S": "u2"}}
	assert.False(t, expr.Evaluate(item2))
}

func TestExpr_BeginsWith(t *testing.T) {
	expr := CompileCondition("begins_with(#sk, :prefix)", map[string]string{"#sk": "sortKey"}, map[string]AttributeValue{":prefix": {"S": "user#"}})
	assert.True(t, expr.Evaluate(Item{"sortKey": {"S": "user#123"}}))
	assert.False(t, expr.Evaluate(Item{"sortKey": {"S": "order#456"}}))
}

func TestExpr_Between(t *testing.T) {
	expr := CompileCondition("#n BETWEEN :lo AND :hi", map[string]string{"#n": "age"}, map[string]AttributeValue{":lo": {"N": "18"}, ":hi": {"N": "65"}})
	assert.True(t, expr.Evaluate(Item{"age": {"N": "30"}}))
	assert.False(t, expr.Evaluate(Item{"age": {"N": "10"}}))
}

func TestExpr_AndOr(t *testing.T) {
	expr := CompileCondition("#a = :v1 AND (#b = :v2 OR #c = :v3)",
		map[string]string{"#a": "x", "#b": "y", "#c": "z"},
		map[string]AttributeValue{":v1": {"S": "a"}, ":v2": {"S": "b"}, ":v3": {"S": "c"}})

	assert.True(t, expr.Evaluate(Item{"x": {"S": "a"}, "y": {"S": "b"}}))
	assert.True(t, expr.Evaluate(Item{"x": {"S": "a"}, "z": {"S": "c"}}))
	assert.False(t, expr.Evaluate(Item{"x": {"S": "a"}, "y": {"S": "nope"}, "z": {"S": "nope"}}))
}

func TestExpr_AttributeExists(t *testing.T) {
	expr := CompileCondition("attribute_exists(#f)", map[string]string{"#f": "field"}, nil)
	assert.True(t, expr.Evaluate(Item{"field": {"S": "val"}}))
	assert.False(t, expr.Evaluate(Item{"other": {"S": "val"}}))
}

func TestExpr_Contains(t *testing.T) {
	expr := CompileCondition("contains(#s, :sub)", map[string]string{"#s": "text"}, map[string]AttributeValue{":sub": {"S": "hello"}})
	assert.True(t, expr.Evaluate(Item{"text": {"S": "say hello world"}}))
	assert.False(t, expr.Evaluate(Item{"text": {"S": "goodbye"}}))
}

func TestExpr_Size(t *testing.T) {
	expr := CompileCondition("size(#s) > :n", map[string]string{"#s": "text"}, map[string]AttributeValue{":n": {"N": "5"}})
	assert.True(t, expr.Evaluate(Item{"text": {"S": "longtext"}}))
	assert.False(t, expr.Evaluate(Item{"text": {"S": "hi"}}))
}

func TestExpr_Update_Set(t *testing.T) {
	item := Item{"pk": {"S": "u1"}, "name": {"S": "Alice"}}
	updated := ApplyUpdate(item, "SET #n = :v", map[string]string{"#n": "name"}, map[string]AttributeValue{":v": {"S": "Bob"}})
	assert.Equal(t, "Bob", updated["name"]["S"])
}

func TestExpr_Update_Remove(t *testing.T) {
	item := Item{"pk": {"S": "u1"}, "name": {"S": "Alice"}, "temp": {"S": "x"}}
	updated := ApplyUpdate(item, "REMOVE #t", map[string]string{"#t": "temp"}, nil)
	_, exists := updated["temp"]
	assert.False(t, exists)
}

func TestExpr_Projection(t *testing.T) {
	item := Item{"pk": {"S": "u1"}, "name": {"S": "Alice"}, "age": {"N": "30"}, "secret": {"S": "x"}}
	projected := ApplyProjection(item, "#n, #a", map[string]string{"#n": "name", "#a": "age"})
	assert.Equal(t, 2, len(projected))
	assert.Equal(t, "Alice", projected["name"]["S"])
	assert.Equal(t, "30", projected["age"]["N"])
}

func TestExpr_Cache(t *testing.T) {
	cache := NewExprCache()
	e1 := cache.GetOrCompile("#pk = :val", map[string]string{"#pk": "id"}, map[string]AttributeValue{":val": {"S": "1"}})
	e2 := cache.GetOrCompile("#pk = :val", map[string]string{"#pk": "id"}, map[string]AttributeValue{":val": {"S": "2"}})
	// Same expression structure, different values — both should work
	assert.True(t, e1.Evaluate(Item{"id": {"S": "1"}}))
	assert.True(t, e2.Evaluate(Item{"id": {"S": "2"}}))
}

func BenchmarkExpr_Evaluate(b *testing.B) {
	expr := CompileCondition("#pk = :val AND #sk > :min",
		map[string]string{"#pk": "userId", "#sk": "timestamp"},
		map[string]AttributeValue{":val": {"S": "u1"}, ":min": {"N": "1000"}})
	item := Item{"userId": {"S": "u1"}, "timestamp": {"N": "2000"}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		expr.Evaluate(item)
	}
}
```

- [ ] **Step 2: Implement expression_ast.go**

Create `services/dynamodb/expression_ast.go` with typed AST nodes:

```go
package dynamodb

// ConditionExpr is a compiled condition that can be evaluated against items.
type ConditionExpr interface {
	Evaluate(item Item) bool
}

// Implementations: andExpr, orExpr, notExpr, compareExpr, betweenExpr,
// beginsWithExpr, containsExpr, attrExistsExpr, attrNotExistsExpr, sizeExpr, inExpr
```

Each node type evaluates directly against the item with typed comparisons — no string parsing in the hot path.

- [ ] **Step 3: Rewrite expression.go**

Replace string-based `evaluateCondition` with `CompileCondition` that returns a `ConditionExpr`. Add `ApplyUpdate` and `ApplyProjection` as exported functions that replace `parseUpdateExpression` and `applyProjection`.

Keep the old function signatures as wrappers that call the new compiled versions, so existing handlers don't break.

- [ ] **Step 4: Run all tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -v`
Expected: All tests PASS

- [ ] **Step 5: Run expression benchmarks**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -bench=BenchmarkExpr -benchmem`
Expected: <50ns/op for simple condition evaluation

- [ ] **Step 6: Commit**

```bash
git add services/dynamodb/expression_ast.go services/dynamodb/expression.go services/dynamodb/expression_test.go
git commit -m "feat(dynamodb): AST-based expression compiler with cache"
```

---

### Task 6: TTL Heap Reaper

**Files:**
- Rewrite: `services/dynamodb/ttl.go`
- Create: `services/dynamodb/ttl_test.go`

- [ ] **Step 1: Write the failing test**

Create `services/dynamodb/ttl_test.go`:

```go
package dynamodb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTTL_ItemExpires(t *testing.T) {
	s := NewTableStore("123456789012", "us-east-1")
	s.CreateTable("ttl-test",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)

	s.UpdateTimeToLive("ttl-test", &TTLSpecification{AttributeName: "expires", Enabled: true})

	// Put item that expires 1 second from now
	expiry := time.Now().Add(1 * time.Second).Unix()
	s.PutItem("ttl-test", Item{
		"pk":      {"S": "temp-item"},
		"expires": {"N": fmt.Sprintf("%d", expiry)},
	})

	// Item should exist immediately
	item, _ := s.GetItem("ttl-test", Item{"pk": {"S": "temp-item"}}, "", nil)
	require.NotNil(t, item)

	// Wait for TTL to fire
	time.Sleep(2 * time.Second)

	// Item should be gone
	item, _ = s.GetItem("ttl-test", Item{"pk": {"S": "temp-item"}}, "", nil)
	assert.Nil(t, item)
}

func TestTTL_NonExpiredItemSurvives(t *testing.T) {
	s := NewTableStore("123456789012", "us-east-1")
	s.CreateTable("ttl-test2",
		[]KeySchemaElement{{AttributeName: "pk", KeyType: "HASH"}},
		[]AttributeDefinition{{AttributeName: "pk", AttributeType: "S"}},
		"PAY_PER_REQUEST", nil, nil, nil, nil)

	s.UpdateTimeToLive("ttl-test2", &TTLSpecification{AttributeName: "expires", Enabled: true})

	// Put item that expires far in the future
	expiry := time.Now().Add(1 * time.Hour).Unix()
	s.PutItem("ttl-test2", Item{
		"pk":      {"S": "keep-item"},
		"expires": {"N": fmt.Sprintf("%d", expiry)},
	})

	time.Sleep(500 * time.Millisecond)

	item, _ := s.GetItem("ttl-test2", Item{"pk": {"S": "keep-item"}}, "", nil)
	assert.NotNil(t, item)
}
```

- [ ] **Step 2: Rewrite ttl.go**

Replace full-scan reaper with heap-based approach:
- `ttlHeap` min-heap ordered by expiry timestamp
- On PutItem: if TTL attribute exists, push to heap and reset timer
- Timer fires at next expiry: pop expired, delete items in batches
- O(log n) per insert, O(1) per expiry check

- [ ] **Step 3: Run tests**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -run TestTTL -v -timeout 30s`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add services/dynamodb/ttl.go services/dynamodb/ttl_test.go
git commit -m "feat(dynamodb): heap-based TTL reaper with timer-driven expiry"
```

---

### Task 7: Wire Store to Handlers + Full Integration Test

**Files:**
- Modify: `services/dynamodb/handlers.go` (minimal changes to use new table methods)
- Modify: `services/dynamodb/service.go` (wire TTL heap)

- [ ] **Step 1: Update handlers to use new table methods**

The handlers call `store.PutItem()`, `store.GetItem()`, etc. — these signatures haven't changed. But internally store.go now delegates to `table.putItem()` etc. Verify all handler round-trips still work.

- [ ] **Step 2: Run the full existing test suite**

Run: `cd /Users/megan/cloudmock && go test ./services/dynamodb/ -v`
Expected: ALL existing tests PASS

- [ ] **Step 3: Run against live CloudMock instance**

Run: `cd /Users/megan/cloudmock && go test -tags smoke ./benchmarks/ -run TestBenchmark_S3_Quick -v`
Expected: PASS (rebuild gateway with new dynamodb code)

- [ ] **Step 4: Commit**

```bash
git add services/dynamodb/
git commit -m "feat(dynamodb): wire new storage engine to handlers"
```

---

### Task 8: DynamoDB Scaling Benchmark

**Files:**
- Create: `benchmarks/suites/stress/dynamodb_stress.go`
- Create: `benchmarks/suites/stress/dynamodb_stress_test.go`

- [ ] **Step 1: Write scaling benchmark**

Create `benchmarks/suites/stress/dynamodb_stress.go` with a stress suite that:
- Creates a table with 1M items
- Benchmarks GetItem, PutItem, Query at that scale
- Reports P50/P95/P99 at 1K, 10K, 100K, 1M items
- Runs 10K concurrent goroutines for mixed read/write

- [ ] **Step 2: Create test**

Create `benchmarks/suites/stress/dynamodb_stress_test.go`:

```go
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
```

- [ ] **Step 3: Run test**

Run: `cd /Users/megan/cloudmock && go test ./benchmarks/suites/stress/ -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add benchmarks/suites/stress/
git commit -m "feat(bench): add DynamoDB scaling stress benchmark"
```

---

### Task 9: Benchmark and Validate Performance Targets

- [ ] **Step 1: Rebuild and restart CloudMock**

```bash
cd /Users/megan/cloudmock && make build-gateway
docker compose -f cloudmock-todo-demo/docker-compose.yml down
docker compose -f cloudmock-todo-demo/docker-compose.yml up -d
```

- [ ] **Step 2: Run benchmark comparison**

```bash
go run ./benchmarks/cmd/bench --target=cloudmock --endpoint=http://localhost:4566 --services=dynamodb --iterations=100 --concurrency=10
```

- [ ] **Step 3: Validate P50 targets**

Expected: GetItem P50 < 0.1ms, PutItem P50 < 0.2ms, Query P50 < 1ms

- [ ] **Step 4: Run Go benchmarks for scaling**

```bash
go test ./services/dynamodb/ -bench=BenchmarkTable_GetItem_1M -benchmem
```

Expected: <100ns/op at 1M items

- [ ] **Step 5: Commit results**

```bash
git add benchmarks/results/
git commit -m "bench: DynamoDB native performance results"
```
